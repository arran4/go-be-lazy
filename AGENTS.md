# Agent Guidelines

## Behavior Changes & Feature Flags

When modifying the behavior of `go-be-lazy`, consider existing users.

*   **Eviction Policies**: The default eviction policy for `MaxSize` is `EvictionPolicyRandom`, which relies on Go's map iteration order. Any new eviction policies should be selectable via options (e.g., `EvictionPolicyFirst`).
    *   **Feature Flag**: Use `WithEvictionPolicy(policy)` to switch behavior.

## Documentation Standards

All public types and functions must be extensively documented. This includes:

*   **Type descriptions**: Explain what the type represents and its purpose.
*   **Field descriptions**: Explain the purpose of each field.
*   **Function descriptions**: Explain what the function does, its parameters, return values, and any side effects (e.g., panic conditions, concurrency guarantees).
*   **Examples**: Provide usage examples for complex functionality where possible.

## Code Conventions

*   **Generic Keys**: All map-related functions must support `[K comparable]` generic keys.
*   **Testing**: Ensure tests cover both generic keys (e.g., `string`, `int`) and specific edge cases (e.g., concurrency, eviction).
