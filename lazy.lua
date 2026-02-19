-- Default lazy.nvim plugin spec for mdp.
-- Users can override any field in their own spec.
return {
  "donaldgifford/mdp",
  main = "mdp",
  ft = "markdown",
  cmd = { "MdpStart", "MdpStop", "MdpToggle", "MdpOpen", "MdpInstall" },
  opts = {},
}
