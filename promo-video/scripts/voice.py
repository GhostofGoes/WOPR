# SPDX-FileCopyrightText: 2026 Christopher Goes
# SPDX-License-Identifier: MIT
"""Generate the promo's voice lines with ElevenLabs, with character timings.

Usage: python3 -I scripts/voice.py [id ...]   (needs ELEVENLABS_API_KEY)
Writes public/audio/vo-<id>.mp3 and audio-src/vo-<id>.json.
"""
import base64, json, os, sys, urllib.request

NARRATOR = "RV8OR7AC0uhWHZnVnmI5"  # Wayne: deep, authoritative trailer voice
WOPR = "onwK4e9ZLuTAKqWW03F9"  # Daniel: steady British voice, processed into a terminal speaker
DRAMATIC = {"stability": 0.42, "similarity_boost": 0.88, "style": 0.38, "use_speaker_boost": True}
FLAT = {"stability": 1.0, "similarity_boost": 0.8, "style": 0.0, "use_speaker_boost": False}

LINES = {
    "n1": (NARRATOR, DRAMATIC, "In nineteen eighty-three... a kid dialed into the wrong computer."),
    "w1": (WOPR, FLAT, "Shall we play a game?"),
    "n2": (NARRATOR, DRAMATIC, "Falken's Maze. Chess. Poker. Fighter Combat."),
    "n3": (NARRATOR, DRAMATIC, "And one game... that cannot be won."),
    "n4": (NARRATOR, DRAMATIC, "Is it a game... or is it real?"),
    "w2": (WOPR, FLAT, "A strange game. The only winning move is not to play."),
    "n5": (NARRATOR, DRAMATIC, "Whopper. Sixteen games. Free, and open source."),
}

def main():
    key = os.environ["ELEVENLABS_API_KEY"]
    os.makedirs("audio-src", exist_ok=True)
    for lid in sys.argv[1:] or LINES:
        voice, settings, text = LINES[lid]
        body = json.dumps({"text": text, "model_id": "eleven_multilingual_v2", "voice_settings": settings}).encode()
        req = urllib.request.Request(
            f"https://api.elevenlabs.io/v1/text-to-speech/{voice}/with-timestamps",
            data=body, headers={"xi-api-key": key, "Content-Type": "application/json"})
        with urllib.request.urlopen(req) as r:
            out = json.load(r)
        raw = "public/audio/vo-%s.mp3" % lid if voice == NARRATOR else "audio-src/raw-%s.mp3" % lid
        with open(raw, "wb") as f:
            f.write(base64.b64decode(out["audio_base64"]))
        al = out["alignment"]
        with open(f"audio-src/vo-{lid}.json", "w") as f:
            json.dump({"text": text, "chars": al["characters"], "start": al["character_start_times_seconds"],
                       "end": al["character_end_times_seconds"]}, f)
        print(lid, "ok", round(al["character_end_times_seconds"][-1], 2), "s")

if __name__ == "__main__":
    main()
