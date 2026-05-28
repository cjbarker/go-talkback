package hotkey

import xhk "golang.design/x/hotkey"

// Preset is a named hotkey combination shown in the menu.
type Preset struct {
	Name string
	Mods []xhk.Modifier
	Key  xhk.Key
}

// Presets is the ordered list of choices shown in the Hotkey submenu.
var Presets = []Preset{
	{Name: "Option+Space", Mods: []xhk.Modifier{xhk.ModOption}, Key: xhk.KeySpace},
	{Name: "Cmd+Shift+Space", Mods: []xhk.Modifier{xhk.ModCmd, xhk.ModShift}, Key: xhk.KeySpace},
	{Name: "Ctrl+Space", Mods: []xhk.Modifier{xhk.ModCtrl}, Key: xhk.KeySpace},
	{Name: "Option+R", Mods: []xhk.Modifier{xhk.ModOption}, Key: xhk.KeyR},
	{Name: "Cmd+Shift+R", Mods: []xhk.Modifier{xhk.ModCmd, xhk.ModShift}, Key: xhk.KeyR},
}

// Find returns the preset matching name, or Presets[0] if not found.
func Find(name string) Preset {
	for _, p := range Presets {
		if p.Name == name {
			return p
		}
	}
	return Presets[0]
}
