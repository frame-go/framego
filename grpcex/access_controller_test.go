package grpcex

import (
	"testing"

	casbin "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
)

const testPolicy = `
p, ServiceA, Cat1Method1
p, ServiceA, Cat1Method2
p, ServiceB, Group1
p, ServiceC, Group2
p, ServiceD, Group3
p, ServiceE, Group1
p, ServiceE, Group2
p, ServiceE, Cat3Method1
p, ServiceF, Cat1*
p, ServiceG, *Method*
p, ServiceH, Group4
p, ServiceH, Cat3Method1

g, Cat1Method1, Group1
g, Cat2Method1, Group1
g, Cat2*, Group2
g, *Method*, Group3
g, Cat1Method1, Group4
g, Cat2*, Group4
`

func initAccessController() (*AccessController, error) {
	m, err := model.NewModelFromString(defaultCasbinModel)
	if err != nil {
		return nil, err
	}
	a := stringadapter.NewAdapter(testPolicy)
	enforcer, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, err
	}
	enforcer.AddNamedMatchingFunc("g", "", globMatch)
	c := &AccessController{
		enforcer: enforcer,
	}
	return c, nil
}

func TestAccessController(t *testing.T) {
	var tests = []struct {
		subject string
		object  string
		result  bool
	}{
		{"ServiceUnknown", "Cate1Method1", false},
		{"ServiceA", "Cat1Method1", true},
		{"ServiceA", "Cat1Method2", true},
		{"ServiceA", "Cat2Method1", false},
		{"ServiceB", "Cat1Method1", true},
		{"ServiceB", "Cat1Method2", false},
		{"ServiceB", "Cat2Method1", true},
		{"ServiceC", "Cat1Method1", false},
		{"ServiceC", "Cat2Method1", true},
		{"ServiceD", "Cat1Method1", true},
		{"ServiceD", "Unknown", false},
		{"ServiceE", "Cat1Method1", true},
		{"ServiceE", "Cat2Method1", true},
		{"ServiceE", "Cat3Method1", true},
		{"ServiceE", "Cat3Method2", false},
		{"ServiceF", "Cat1Method1", true},
		{"ServiceF", "Cat2Method1", false},
		{"ServiceG", "Cat1Method1", true},
		{"ServiceG", "Unknown", false},
		{"ServiceH", "Cat1Method1", true},
		{"ServiceH", "Cat1Method2", false},
		{"ServiceH", "Cat2Method1", true},
		{"ServiceH", "Cat3Method1", true},
		{"ServiceH", "Cat3Method2", false},
	}
	c, err := initAccessController()
	if err != nil {
		t.Errorf("init_access_controller_error: %v", err)
		return
	}
	for _, test := range tests {
		if c.checkServiceAccess(test.subject, test.object) != test.result {
			t.Errorf("check_access_error: %v", test)
		}
	}
}

func BenchmarkAccessController(b *testing.B) {
	c, err := initAccessController()
	if err != nil {
		b.Errorf("init_access_controller_error: %v", err)
		return
	}
	b.Run("simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.checkServiceAccess("ServiceA", "Cat1Method1")
		}
	})
	b.Run("complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.checkServiceAccess("ServiceH", "Cat2Method1")
		}
	})
}
