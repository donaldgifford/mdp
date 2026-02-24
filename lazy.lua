-- Default lazy.nvim plugin spec for mdp.
-- Users can override any field in their own spec.
return {
  "donaldgifford/mdp",
  main = "mdp",
  ft = "markdown",
  cmd = { "MdpPreview", "MdpStart", "MdpStop", "MdpToggle", "MdpOpen", "MdpInstall" },
  keys = {
    { "<leader>mp", "<cmd>MdpPreview<cr>", ft = "markdown", desc = "Markdown preview" },
  },
  opts = {},
}
