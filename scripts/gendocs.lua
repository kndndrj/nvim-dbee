local docgen = require("docgen")

local docs = {}

docs.test = function()
  -- Filepaths that should generate docs
  local input_files = {
    "./lua/dbee/types_doc.lua",
    "./lua/dbee/api/core.lua",
    "./lua/dbee/api/tiles.lua",
  }

  -- Output file
  local output_file = "./doc/dbee-api.txt"
  local output_file_handle = assert(io.open(output_file, "w"), "failed opening file for writing")

  for _, input_file in ipairs(input_files) do
    docgen.write(input_file, output_file_handle)
  end

  output_file_handle:write(" vim:tw=78:ts=8:ft=help:norl:\n")
  output_file_handle:close()
  vim.cmd("checktime")
end

docs.test()

return docs
