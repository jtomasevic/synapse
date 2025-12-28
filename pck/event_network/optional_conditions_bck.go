package event_network

//type AdditionalCondition struct {
//	MaxDepth        *int
//	Within          *int
//	WithinMeasure   *TimeUnit
//	Count           *int
//	CountOrMore     bool
//	AdditionalTypes []string
//	PropertyValues  map[string]any
//}
//
//type Condition func(*AdditionalCondition)
//
//func WithMaxDepth(maxDepth int) Condition {
//	return func(c *AdditionalCondition) {
//		c.MaxDepth = &maxDepth
//	}
//}
//
//func Within(withing int, measure TimeUnit) Condition {
//	return func(c *AdditionalCondition) {
//		c.Within = &withing
//		c.WithinMeasure = &measure
//	}
//}
//
//func WithCount(count int, orMode bool) Condition {
//	return func(c *AdditionalCondition) {
//		c.Count = &count
//		c.CountOrMore = orMode
//	}
//}
//
//func OrType(types ...string) Condition {
//	return func(c *AdditionalCondition) {
//		c.AdditionalTypes = append(c.AdditionalTypes, types...)
//	}
//}
//
//func WithPropertyValue(name string, value any) Condition {
//	return func(c *AdditionalCondition) {
//		if c.PropertyValues == nil {
//			c.PropertyValues = make(map[string]any)
//		}
//		c.PropertyValues[name] = value
//	}
//}
