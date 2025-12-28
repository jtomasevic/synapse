package event_network

import "fmt"

func printDerivedFrom(net *InMemoryEventNetwork, ev Event, indent string) {
	edges := net.in[ev.ID] // contributors → ev
	if len(edges) == 0 {
		return
	}

	for _, edge := range edges {
		c := net.events[edge.From]
		fmt.Printf(
			"%s↳ %s (%s)\n",
			indent,
			c.EventType,
			c.ID.String()[:8],
		)
	}
}

func PrintEventGraph(network EventNetwork) {
	net := network.(*InMemoryEventNetwork)
	levels := computeDerivationLevels(net)

	grouped := make(map[int][]Event)
	maxLevel := 0

	for id, lvl := range levels {
		grouped[lvl] = append(grouped[lvl], net.events[id])
		if lvl > maxLevel {
			maxLevel = lvl
		}
	}

	for lvl := maxLevel; lvl >= 0; lvl-- {
		fmt.Printf("\n[Level %d]\n", lvl)

		events := grouped[lvl]
		for i, ev := range events {
			prefix := "├──"
			if i == len(events)-1 {
				prefix = "└──"
			}

			fmt.Printf(
				"%s %s (%s)\n",
				prefix,
				ev.EventType,
				ev.ID.String()[:8],
			)

			printDerivedFrom(net, ev, "    ")
		}
	}
}

func computeDerivationLevels(net *InMemoryEventNetwork) map[EventID]int {
	levels := make(map[EventID]int)
	visited := make(map[EventID]bool)

	var dfs func(EventID) int
	dfs = func(id EventID) int {
		if visited[id] {
			return levels[id]
		}
		visited[id] = true

		children := net.in[id] // semantic children (contributors)
		if len(children) == 0 {
			levels[id] = 0 // leaf event
			return 0
		}

		maxChildLevel := 0
		for _, e := range children {
			cl := dfs(e.From)
			if cl > maxChildLevel {
				maxChildLevel = cl
			}
		}

		levels[id] = maxChildLevel + 1
		return levels[id]
	}

	for id := range net.events {
		dfs(id)
	}

	return levels
}
