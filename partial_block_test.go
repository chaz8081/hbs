package handlebars

import "testing"

var partialBlockTests = []Test{
	{"basic partial block", `{{#> layout}}content{{/layout}}`, nil, nil, nil, map[string]string{"layout": "before-{{> @partial-block}}-after"}, "before-content-after"},
	{"fallback when partial missing", `{{#> nonexistent}}fallback content{{/nonexistent}}`, nil, nil, nil, nil, "fallback content"},
	{"partial block with context", `{{#> layout}}Hello {{name}}{{/layout}}`, map[string]string{"name": "World"}, nil, nil, map[string]string{"layout": "<div>{{> @partial-block}}</div>"}, "<div>Hello World</div>"},
	{"nested partial blocks", `{{#> outer}}inner content{{/outer}}`, nil, nil, nil, map[string]string{"outer": "OUTER[{{#> middle}}{{> @partial-block}}{{/middle}}]OUTER", "middle": "MIDDLE[{{> @partial-block}}]MIDDLE"}, "OUTER[MIDDLE[inner content]MIDDLE]OUTER"},
	{"content around @partial-block", `{{#> page}}My Page Content{{/page}}`, nil, nil, nil, map[string]string{"page": "<html><body>{{> @partial-block}}</body></html>"}, "<html><body>My Page Content</body></html>"},
	{"multiple @partial-block refs", `{{#> doubled}}hello{{/doubled}}`, nil, nil, nil, map[string]string{"doubled": "{{> @partial-block}} and {{> @partial-block}}"}, "hello and hello"},
	{"empty block content", `{{#> layout}}{{/layout}}`, nil, nil, nil, map[string]string{"layout": "[{{> @partial-block}}]"}, "[]"},
	{"partial ignores block content", `{{#> simple}}this is ignored{{/simple}}`, nil, nil, nil, map[string]string{"simple": "just a simple partial"}, "just a simple partial"},
	{"partial block with context param", `{{#> myPartial ctxData}}block content {{foo}}{{/myPartial}}`, map[string]interface{}{"ctxData": map[string]string{"foo": "bar"}}, nil, nil, map[string]string{"myPartial": "{{foo}}: {{> @partial-block}}"}, "bar: block content bar"},
}

func TestPartialBlocks(t *testing.T) { launchTests(t, partialBlockTests) }
