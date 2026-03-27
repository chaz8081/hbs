package handlebars

import (
	"testing"
)

func TestDataVar_Root(t *testing.T) {
	tpl := MustParse("{{#each items}}{{@root.title}}: {{this}}{{/each}}")
	ctx := map[string]interface{}{
		"title": "List",
		"items": []string{"a", "b"},
	}
	result := tpl.MustExec(ctx)
	if result != "List: aList: b" {
		t.Errorf("expected 'List: aList: b', got %q", result)
	}
}

func TestDataVar_Index(t *testing.T) {
	tpl := MustParse("{{#each items}}{{@index}}:{{this}} {{/each}}")
	ctx := map[string]interface{}{"items": []string{"a", "b", "c"}}
	result := tpl.MustExec(ctx)
	if result != "0:a 1:b 2:c " {
		t.Errorf("expected '0:a 1:b 2:c ', got %q", result)
	}
}

func TestDataVar_Key(t *testing.T) {
	tpl := MustParse("{{#each obj}}{{@key}}={{this}} {{/each}}")
	ctx := map[string]interface{}{"obj": map[string]interface{}{"x": "1"}}
	result := tpl.MustExec(ctx)
	if result != "x=1 " {
		t.Errorf("expected 'x=1 ', got %q", result)
	}
}

func TestDataVar_First(t *testing.T) {
	tpl := MustParse("{{#each items}}{{#if @first}}FIRST{{/if}}{{this}} {{/each}}")
	ctx := map[string]interface{}{"items": []string{"a", "b"}}
	result := tpl.MustExec(ctx)
	if result != "FIRSTa b " {
		t.Errorf("expected 'FIRSTa b ', got %q", result)
	}
}

func TestDataVar_Last(t *testing.T) {
	tpl := MustParse("{{#each items}}{{this}}{{#if @last}}!{{/if}} {{/each}}")
	ctx := map[string]interface{}{"items": []string{"a", "b"}}
	result := tpl.MustExec(ctx)
	if result != "a b! " {
		t.Errorf("expected 'a b! ', got %q", result)
	}
}

func TestDataVar_RootInNestedEach(t *testing.T) {
	tpl := MustParse("{{#each outer}}{{#each this}}{{@root.title}}{{/each}}{{/each}}")
	ctx := map[string]interface{}{
		"title": "T",
		"outer": [][]string{{"a"}, {"b"}},
	}
	result := tpl.MustExec(ctx)
	if result != "TT" {
		t.Errorf("expected 'TT', got %q", result)
	}
}
