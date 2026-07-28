# What this run certifies

`640db264` fixed an unsigned wrap: `before.Alloc - after.Alloc` on two `uint64`
values underflows to a number near 2^64 whenever the heap GREW across the
measured window, so the "memory freed" diagnostic could report roughly 1.8e13 MB
instead of a small negative delta. The subtraction now converts to `float64`
first, in a single named helper shared by the GUI and no-GUI entry points.

Captured here: the helper's signature in the shipped source, and all four of the
commit's boundary guards executed for real (grow / shrink / no-change /
formatting) with `--- PASS:` asserted per test, plus the package under `-race`.
