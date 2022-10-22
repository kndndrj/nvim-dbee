-- Show error codes in the output
codes = true

-- Disable unused argument warning for "self"
self = false

ignore = {
  "122", -- Indirectly setting a readonly global
  "631", -- line too long
}

-- Per file ignores
files["lua/projector/contract/*"] = { ignore = { "212" } } -- Ignore unused argument warning for interfaces

-- Global objects defined by the C code
read_globals = {
  "vim",
}
