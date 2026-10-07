# Metrology Assurance Suite

## Project Goal

Metrology Assurance Suite is a local HTTP service that keeps a traceable record
of laboratory measurement assurance work. It registers devices, opens a
verification case for one device, records technician measurements, requests a
repeat measurement when tolerance is exceeded, and seals the final evidence
record after review.

The service is intentionally self-contained. It uses an in-memory repository
with atomic copy-on-write mutations and does not require a database or an
external service.

## Users

- Laboratory steward: registers devices and opens verification cases.
- Measurement technician: records observed values for an open case.
- Independent confirmer: a second person who confirms a specific accepted
  measurement on a high-risk case; the confirmer must differ from the
  technician who recorded that measurement.
- Quality verifier: reviews accepted measurements and seals the case.

## Core Entities

- Device: a laboratory asset with a unique asset tag, model, room, steward, and
  lifecycle state.
- VerificationCase: one bounded assurance activity for a device, with a nominal
  value, tolerance, risk level, current state, version, and assigned steward.
- Measurement: an observed value, computed deviation, acceptance decision,
  technician, and timestamp.
- Confirmation: an immutable record that a second person independently
  confirmed one specific accepted measurement, including the confirmer, the
  basis for the confirmation, and the timestamp.
- EvidenceSeal: the immutable closing record for a reviewed case, including the
  selected accepted measurement and, for high-risk cases, the confirmation
  that authorizes it.
- AuditEvent: an append-only trace of meaningful case and device changes.

## Workflows

### 1. Register Device

The steward sends `POST /devices` with an asset tag, model, room, and steward.
The service validates the request, normalizes the asset tag, rejects duplicate
tags, and creates an active device.

### 2. Open Verification Case

The steward sends `POST /cases` with a device ID, nominal value, tolerance,
owner, and an optional risk level (`standard` by default, or `high`). The
service verifies that the device exists, is active, and has no unfinished
case. A valid request creates a case in `awaiting_measurement`. The risk
level is fixed at creation and cannot be changed afterwards.

### 3. Record Measurement

A technician sends `POST /cases/{id}/measurements`. The case must be
`awaiting_measurement`. The service computes the signed deviation from the
nominal value. An in-tolerance measurement moves the case to
`review_ready`; an out-of-tolerance measurement moves it to
`recheck_required`.

### 4. Confirm Measurement

A second person sends `POST /cases/{id}/confirmations` with a measurement ID,
a confirmer, and the basis for the confirmation. The measurement must belong
to the case and be accepted, the case must not be sealed, and the confirmer
must differ from the technician who recorded the measurement (compared
case-insensitively). A measurement carries at most one confirmation: a
duplicate request is rejected and the original record stays untouched.
Confirmations are append-only; there is no update or delete path, so later
personnel or requirement changes cannot rewrite an existing confirmation.

### 5. Reopen Verification Case

The steward sends `POST /cases/{id}/reopen` after an out-of-tolerance result.
Only `recheck_required` cases may move back to `awaiting_measurement`. The
service records the actor and reason in the audit trail.

### 6. Seal Verification

The verifier sends `POST /cases/{id}/seal` after an accepted measurement. Only
`review_ready` cases may be sealed. The service selects the accepted
measurement with the smallest absolute deviation. A high-risk case also
requires an independent confirmation of that measurement; without one the
seal is rejected with `confirmation_required` and nothing is committed. The
service creates exactly one evidence seal — citing the confirmation for
high-risk cases — moves the case to `sealed`, and rejects later mutations.

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
- A case can have at most one evidence seal.
- A sealed case cannot accept measurements, confirmations, reopen, or seal
  again.
- A confirmation targets exactly one accepted measurement of its case.
- A measurement has at most one confirmation; the confirmer must differ from
  the recording technician.
- Confirmations are immutable once recorded.
- A high-risk case can be sealed only when the selected measurement carries
  an independent confirmation, and the seal cites that confirmation.
- Every mutation that changes a case also appends an audit event in the same
  repository commit; a rejected confirmation or seal commits nothing.

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
seal. The confirmation check rejects self-confirmation, duplicate and
concurrent confirmations, and sealing an unconfirmed high-risk case, and
verifies that each rejected attempt commits nothing.

Tests are intentionally deferred for this initialization baseline. The later
engineering-task stage will add unit and integration tests, including red/green
verification for the workflows and their failure paths.
