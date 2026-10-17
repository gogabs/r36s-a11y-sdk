package a11y

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserVoiceReadsConfigJSONAndOldFile(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"config.json": "{\n  \"system\": {\n    \"SCREEN\": \"1\"\n  },\n  \"voice\": {\n    \"PITCH\": 40,\n    \"RATE\": 70,\n    \"VOICE\": \"Letícia-F123\",\n    \"VOICE_MODULE\": \"rhvoice\"\n  }\n}\n",
		"voz.conf":    "RATE=70\nPITCH=40\nVOICE_MODULE='rhvoice'\nVOICE='Letícia-F123'\n",
	}
	for name, text := range cases {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("A11Y_VOZ_CONF", path)
		numbers, names := userVoice()
		if numbers["RATE"] != 40 || numbers["PITCH"] != -20 || names["VOICE"] != "Letícia-F123" || names["VOICE_MODULE"] != "rhvoice" {
			t.Errorf("%s: %v %v", name, numbers, names)
		}
	}
}
