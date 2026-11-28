# Registry-backed MCP invocation

`Client.Call` uses a cached dubbo-go generic reference scoped to the mesh,
registry, registration mode, service identity, and the binding's fixed `dubbo`
or `tri` protocol. The SDK directory and router choose the Provider. Each
call has a context deadline. Triple non-IDL generic calls use Hessian2, so
the Provider must support that serialization.
Dubbo's pinned SDK can return after the context deadline; the client enforces
the same deadline on its result and treats a late reply as `unknown-outcome`.

| Outcome | `dubbo` | `tri` |
| --- | --- | --- |
| `not-executed` | Invalid input, no Provider, or unsafe effective governance before dispatch | Same |
| `remote-error` | Decoded generic Provider exception | Server-sent status other than cancellation, deadline, or unavailability |
| `unknown-outcome` | Timeout, disconnect, or an error without a decoded Provider exception after dispatch | Timeout, disconnect, cancellation, unavailability, or an unconfirmed local transport error after dispatch |

`unknown-outcome` means the Provider may already have completed the operation.
The caller must not retry automatically. Triple errors do not carry the Java
cause chain or stack trace available in some Dubbo exception payloads.

The safety cluster checks the effective reference and every routed Provider
candidate, then uses dubbo-go's configured load balancer and invokes only its
selected candidate. It rejects unsafe cluster or retry overrides without
dispatching and bypasses cluster interceptors that could add another attempt.
Application-level and interface-level registry modes each use their native SDK
directory; the client does not select a Provider URL.
