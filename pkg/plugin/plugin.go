package plugin

import "fmt"

type Plugin struct {
	Type string
	Name string
	Repo string
}

func (p *Plugin) GetName() string {
	return fmt.Sprintf("%s-%s", p.Type, p.Name)
}
