package streaming

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"html/template"
	"net/http"
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
					<td>{{ .Stream.Stream.FileLocator }}:{{ .Stream.Stream.StreamId }} ({{ .Stream.Stream.StreamType }}) </td>
					<td style="width: 200px;">{{ .Stream.Representation.RepresentationId }}</td>
					<td>{{ .OutputDir }}</td>
					<td>{{ .State | sessionStateText }}</td>
					<td>{{ .ProgressPercentage }}%</td>
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
	var playbackSessions = PBSManager.GetPlaybackSessions()

	for _, s := range playbackSessions {
		transcodingSessions = append(transcodingSessions, s.TranscodingSession)
	}

	templateData := map[string]interface{}{
		"sessions": playbackSessions,
	}

	t := template.Must(template.New("manifest").
		Funcs(map[string]any{
			"sessionStateText": func(state ffmpeg.SessionState) string {
				if stateText, exists := ffmpeg.StateToString[state]; exists {
					return stateText
				}
				return fmt.Sprintf("Unknown State: %v", state)
			},
		}).
		Parse(transcodingSessionsDebugPageTemplate))

	err := t.Execute(w, templateData)
	if err != nil {
		log.WithError(err).Error("error executing HTML template")
	}
}
