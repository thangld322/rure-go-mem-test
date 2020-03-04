Reproduce memleak in rure-go

I have 20000 nginx access raw plaintext logs and 8 patterns to match these logs.
I compiled each pattern once and appended it to a slice of *rure.Regex to use later.
Finally, I loop over all 20000 logs and call `Captures()` to see whether it matches any pattern.