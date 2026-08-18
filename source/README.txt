Original/reference Gun Mayhem Flash/FFDec export used to verify and continue the source-faithful Go port.

This tree is intentionally tracked in Git. It is not a runtime dependency of the Go game: the port reads only from assets/.

Some data can exist both here and in assets/. That duplication is intentional. source/ preserves reference material and assets/ contains only the files the current Go runtime needs.

Do not remove source/ when preparing the repository; it is part of the reverse-engineering/reference material for the project.
