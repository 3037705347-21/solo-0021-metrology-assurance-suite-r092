# Metrology Assurance Suite

## Project Goal

Metrology Assurance Suite is a local HTTP service that keeps a traceable record
of laboratory measurement assurance work. It registers devices, opens a
verification case for one device at a fixed risk level, records technician
measurements, requests a repeat measurement when tolerance is exceeded, records
independent second-person confirmations for high-risk cases, and seals the
final evidence record after review.

The service is intentionally self-contained. It uses an in-memory repository
with atomic copy-on-write mutations and does not require a database or an
external service.

## Users

- Laboratory steward: registers devices and opens verification cases.
- Measurement technician: records observed values for an open case.
- Independent confirmer: a second person who confirms the key measurement of a
  high-risk case before it can be sealed.
- Quality verifier: reviews accepted measurements and seals the case.

## Core Entities

- Device: a laboratory asset with a unique asset tag, model, room, steward, and
  lifecycle state.
- VerificationCase: one bounded assurance activity for a device, with a nominal
  value, tolerance, fixed risk level, current state, version, and assigned
  steward.
- Measurement: an observed value, computed deviation, acceptance decision,
  technician, and timestamp.
- IndependentConfirmation: an append-only record that a confirmer independent
  of the recording technician checked a specific accepted measurement, with the
  confirmation basis and a snapshot of the risk requirement in effect.
- EvidenceSeal: the immutable closing record for a reviewed case, including the
  selected accepted measurement and the confirmation records that satisfied the
  risk gate.
- AuditEvent: an append-only trace of meaningful case and device changes.

## Workflows

### 1. Register Device

The steward sends `POST /devices` with an asset tag, model, room, and steward.
The service validates the request, normalizes the asset tag, rejects duplicate
tags, and creates an active device.

### 2. Open Verification Case

The steward sends `POST /cases` with a device ID, nominal value, tolerance,
owner, and risk level (`standard` or `high`, defaulting to `standard`). The
service verifies that the device exists, is active, and has no unfinished case.
A valid request creates a case in `awaiting_measurement`. The risk level is
fixed for the case lifetime and determines how many distinct independent
confirmations the sealed measurement must have (zero for standard risk, one
for high risk).

### 3. Record Measurement

A technician sends `POST /cases/{id}/measurements`. The case must be
`awaiting_measurement`. The service computes the signed deviation from the
nominal value. An in-tolerance measurement moves the case to
`review_ready`; an out-of-tolerance measurement moves it to
`recheck_required`.

### 4. Reopen Verification Case

The steward sends `POST /cases/{id}/reopen` after an out-of-tolerance result.
Only `recheck_required` cases may move back to `awaiting_measurement`. The
service records the actor and reason in the audit trail.

### 5. Independently Confirm Measurement

For a high-risk case, a confirmer sends
`POST /cases/{id}/confirmations` with an accepted measurement ID, the
confirmer identity, and the confirmation basis (for example a procedure or
traceability reference). The confirmer must be a different person from the
technician who recorded the measurement. The same confirmer cannot confirm the
same measurement twice. Confirmations are only accepted while the case is
`review_ready` and are never accepted for a standard-risk case. A rejected
attempt commits nothing.

### 6. Seal Verification

The verifier sends `POST /cases/{id}/seal` after an accepted measurement. Only
`review_ready` cases may be sealed. The service selects the accepted
measurement with the smallest absolute deviation, then enforces the risk gate:
a high-risk case needs the required number of distinct independent
confirmations of exactly that measurement. If the gate is not satisfied the
seal is rejected and nothing is written. On success the service creates exactly
one evidence seal that references the satisfying confirmation records, moves
the case to `sealed`, and rejects later mutations.

## State And Rules

Device state:

- `active -> retired`
- `retired` is terminal.

VerificationCase state:

- `awaiting_measurement -> review_ready`
- `awaiting_measurement -> recheck_required`
- `recheck_required -> awaiting_measurement`
- `review_ready -> sealed`
- `sealed` is terminal.

Required invariants:

- Asset tags are unique after case-insensitive normalization.
- A device has at most one unfinished verification case.
- Measurements are accepted only while a case is awaiting measurement.
- A tolerance must be positive.
- A case risk level is fixed at creation and never changes afterwards.
- A confirmation requires a confirmer other than the recording technician, a
  non-empty basis, and an accepted measurement on a high-risk case.
- A confirmation is immutable once committed: it snapshots the confirmer,
  technician, basis, measurement, risk level, required count, and policy
  version, so later personnel or requirement changes cannot rewrite it.
- One confirmer cannot confirm the same measurement twice; distinct
  confirmations are counted by normalized confirmer identity.
- A high-risk case cannot be sealed until the selected measurement has the
  required number of distinct independent confirmations; a standard-risk case
  never stores confirmations.
- The seal records the IDs of the confirmations that satisfied the gate.
- A case can have at most one evidence seal.
- A sealed case cannot accept measurements, confirmations, reopen, or seal
  again.
- Every mutation that changes a case also appends an audit event in the same
  repository commit; a failed attempt commits no partial state.

## Modules

- `cmd/metrology-service`: process entrypoint and graceful HTTP shutdown.
- `cmd/workflowcheck`: bounded smoke checks that call the public HTTP API.
- `internal/httpapi`: routing, request decoding, and stable error mapping.
- `internal/assurance`: workflow orchestration and atomic decisions.
- `internal/domain`: entities, validation, state transitions, and errors.
- `internal/repository`: mutex-protected in-memory persistence with copy-on-write
  commits.
- `internal/contracts`: request and response contracts shared by the HTTP layer.

## Public Interfaces

- `GET /healthz`
- `POST /devices`
- `GET /devices/{id}`
- `POST /cases`
- `GET /cases/{id}`
- `POST /cases/{id}/measurements`
- `POST /cases/{id}/confirmations`
- `POST /cases/{id}/reopen`
- `POST /cases/{id}/seal`
- `GET /cases/{id}/events`

## Validation Plan

The baseline build command is `go build ./...`. Each declared workflow has a
bounded production smoke command under `cmd/workflowcheck`. The command starts
an HTTP test server in the same process, calls the public routes, and verifies
both success and failure behavior. The seal check also races two concurrent
seal requests to prove that the repository mutation lock permits exactly one
seal. The confirmation check covers the unsatisfied gate, the same-person and
duplicate rejections, the requirement and policy snapshot, the standard-risk
rejection, and sealed-case immutability; the seal check additionally races a
confirmation against a seal over many rounds to prove an unsatisfied gate can
never seal.

Tests are intentionally deferred for this initialization baseline. The later
engineering-task stage will add unit and integration tests, including red/green
verification for the workflows and their failure paths.
