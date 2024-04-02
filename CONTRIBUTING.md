# Contributing

Pull requests are welcomed. You must

- Sign the Elastic [Contributor License Agreement](https://www.elastic.co/contributor-agreement).
- Include a changelog entry at `.changelog/{pr-number}.txt` with your pull
  request. Or label the PR with `skip-changelog` if it is a non-user facing
  change (test fixes, CI changes, etc.).
- Include tests that demonstrate the change is working.

The `.changelog/{pr-number}.txt` changelog file must follow a Markdown code
block format like

~~~
```release-note:enhancement
webook: Added support for HTTP response headers.
```
~~~

You must use one of these types:

- `release-note:enhancement`
- `release-note:bug`
- `release-note:deprecation`
- `release-note:breaking-change`
- `release-note:new-resource`

The changelog file may contain more than one Markdown code block if there is
more than one change.

~~~
```release-note:enhancement
http mock server: Added minify_json template helper function for minifying static JSON.
```

```release-note:bugfix
http mock server: Fixed YAML unmarshaling of numbers.
```
~~~

## Releasing

To create a new release, use the release workflow in GitHub actions. This will create a new draft
release in GitHub releases with a changelog. After the job completes, review the draft and if
everything is correct, publish the release. When the release is published, GitHub will create the
git tag.
