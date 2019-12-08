package streaming

import (
	"html/template"
	"net/http"

	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/ffmpeg"
)

const transcodingSessionsDebugPageTemplate = `
<html>
	<head><style>
		table, th, td {
		  border: 1px solid black;
		  border-collapse: collapse;
		}
		td {
			padding: 5px;
		}
	</head></style>
	<body>
		<table style="border: 1px solid black;">
			<caption>Sessions</caption>
			<thead><tr>
				<th>Source</th>
				<th>Target Representation</th>
				<th>Directory</th>
				<th>Status</th>
				<th>Progress</th>
			</tr></thead>
			<tbody>
			{{ range .sessions }}
				<tr>
					{{ with .TranscodingSession }}
					<td>{{ .Stream.Stream.MediaFileURL }}:{{ .Stream.Stream.StreamId }} ({{ .Stream.Stream.StreamType }}) </td>
					<td style="width: 200px;">{{ .Stream.Representation.RepresentationId }}</td>
					<td>{{ .OutputDir }}</td>
					<td>{{if .Terminated }}
						Terminated
						{{ else }} {{if .Throttled }}Throttled{{else}}Full Steam!{{end}}
						{{end}}</td>
					<td>{{ printf  "%.1f" .ProgressPercent }}%</td>
					{{ end }}
				</tr>
			{{ end }}
			</tbody>
		</table>
	</body>
</html>
`

func servePlaybackSessionDebugPage(w http.ResponseWriter, r *http.Request) {
	if !viper.GetBool("debug.streamingPages") {
		http.Error(w, "Debug pages not enabled", http.StatusForbidden)
		return
	}

	var transcodingSessions []*ffmpeg.TranscodingSession

	for _, s := range playbackSessions {
		transcodingSessions = append(transcodingSessions, s.TranscodingSession)
	}

	templateData := map[string]interface{}{
		"sessions": playbackSessions,
	}

	t := template.Must(template.New("manifest").Parse(transcodingSessionsDebugPageTemplate))
	t.Execute(w, templateData)
}
