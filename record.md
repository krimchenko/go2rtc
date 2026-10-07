# Recording path template

`record.basePath` is a Go `text/template` evaluated for every new segment in the configured `record.timezone`. If the timezone is missing, empty, or unknown, the system timezone is used.

```yaml
record:
  basePath: '/mnt/recordings/{{.YEAR}}/{{.MONTH}}/{{.DAY}}/{{.ID}}'
```

Stream fields are `{{.ID}}` (the key under `streams`), `{{.URL}}` (the stream URL), and `{{.DEVICE_NAME}}` (the optional `device_name` value). A missing or empty `device_name` expands to an empty string.

Date and time fields are `{{.DATETIME}}` (`YYYY-MM-DD_HH-MM-SS`), `{{.DATE}}` (`YYYY-MM-DD`), `{{.TIME}}` (`HH-MM-SS`), and the zero-padded components `{{.YEAR}}`, `{{.MONTH}}`, `{{.DAY}}`, `{{.HOUR}}`, `{{.MINUTE}}`, and `{{.SECOND}}`. `{{.TIMESTAMP}}` is Unix time in seconds. For another format, use `{{.NOW.Format "2006/01/02/15/04/05"}}` with Go's time layout syntax.

`{{.DURATION}}` is the configured segment duration (for example, `10s`). `{{.NUMBER}}` is the segment number for this stream, starting at `00000001` when the recorder starts and padded to at least eight digits. All variables are available in both `basePath` and `filename` templates.

`record.segmentDuration` accepts values such as `10s` or `10`; a value without a unit is interpreted as seconds for calculations. `{{.DURATION}}` retains the configured value (`10` stays `10`).

Keep a fixed directory before the first template expression, such as `/mnt/recordings/`. Background finalization and cleanup search from this directory. If `basePath` has no template expression, the stream ID is appended as a directory, as before.
