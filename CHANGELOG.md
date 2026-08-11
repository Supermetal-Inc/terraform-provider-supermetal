## v0.2.0

Requires Supermetal agent v0.1.12 or newer.

- Add the complete connector envelope: `buffer`, `identifier_naming`, `limits`, and `schedule`.
- Add File sink and Redshift sink support from the v0.1.11 OpenAPI specification.
- Add MongoDB majority threshold, list-valued migration strategy, recursive union validation, and connector-specific immutable field support.
- Preserve redacted free-form secrets without retaining ordinary deleted map entries.
- Emit explicit nulls when Terraform removes optional secrets so the agent can distinguish removal from redaction.
- Allow catalog maps whose keys are unknown during planning.
- Validate buffer configuration before create and update.
- Fix stale state when optional blocks are removed or union variants change.
- Preserve null-versus-empty semantics when the API defaults optional maps such as `buffer.options` to an empty collection.

## v0.1.0

Initial release. Requires agent >= 0.1.10.
