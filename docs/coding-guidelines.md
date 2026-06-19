# Coding Guidelines

## Naming

Names must reveal intent.

Bad:

```go
Process()
Handle()
Execute()
DoStuff()
```

Good:

```go
CreateOrder()
ReserveInventory()
ValidatePayment()
PublishOrderCreatedEvent()
```

---

# Functions

Preferred:

```text
20-40 lines
```

Maximum:

```text
80 lines
```

If larger, refactor.

---

# Nesting

Maximum:

```text
3 levels
```

Use early returns.

Bad:

```go
if a {
	if b {
		if c {
		}
	}
}
```

Good:

```go
if !a {
	return
}

if !b {
	return
}

if !c {
	return
}
```

---

# Comments

Avoid explaining WHAT.

Allowed comments:

* Why
* Business Rules
* Architectural Decisions
* Non-obvious Constraints

---

# Package Design

Each package should have a clear purpose.

Avoid:

```text
utils/
helpers/
common/
misc/
```

Prefer domain-specific packages.

---

# Interfaces

Create interfaces only when:

* Multiple implementations exist
* Testing requires abstraction
* Boundary contracts are needed

Do not create interfaces prematurely.

---

# Database Access

Business logic must never contain SQL.

Bad:

```go
func CreateOrder() {
	db.Exec(...)
}
```

Good:

```go
func CreateOrder() {
	orderRepo.Save(...)
}
```

---

# API Handlers

Handlers must:

1. Parse request
2. Validate request
3. Invoke use case
4. Return response

Handlers must not:

* Query database
* Publish Kafka events
* Contain business rules

---

# Kafka Consumers

Consumers must:

1. Deserialize event
2. Validate event
3. Call use case

Nothing else.

---

# Code Reviews

Every change should be reviewed for:

* Readability
* Maintainability
* Scalability
* Security
* Testability

Optimization comes after correctness.
