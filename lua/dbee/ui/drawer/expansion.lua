local M = {}

-- applies the expansion on new nodes
---@param tree NuiTree tree to apply the expansion map to
---@param expansion table<string, boolean> expansion map ( id:is_expanded mapping )
function M.set(tree, expansion)
  -- first pass: load lazy_loaded children
  for id, t in pairs(expansion) do
    if t then
      local node = tree:get_node(id) --[[@as DrawerUINode]]
      if node then
        -- if function for getting layout exist, call it
        if type(node.lazy_children) == "function" then
          tree:set_nodes(node.lazy_children(), node.id)
        end
      end
    end
  end

  -- second pass: expand nodes
  for id, t in pairs(expansion) do
    if t then
      local node = tree:get_node(id) --[[@as DrawerUINode]]
      if node then
        node:expand()
      end
    end
  end
end

-- gets an expansion config to restore the expansion on new nodes
---@param tree NuiTree
---@return table<string, boolean>
function M.get(tree)
  ---@type table<string, boolean>
  local nodes = {}

  local function process(node)
    if node:is_expanded() then
      nodes[node:get_id()] = true
    end

    if node:has_children() then
      for _, n in ipairs(tree:get_nodes(node:get_id())) do
        process(n)
      end
    end
  end

  for _, node in ipairs(tree:get_nodes()) do
    process(node)
  end

  return nodes
end

return M
