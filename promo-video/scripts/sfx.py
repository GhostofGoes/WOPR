# SPDX-FileCopyrightText: 2026 Christopher Goes
# SPDX-License-Identifier: MIT
"""Generate the promo's sound effects and score with ElevenLabs.

Usage: python3 -I scripts/sfx.py [name ...]   (needs ELEVENLABS_API_KEY)
Writes public/audio/sfx-<name>.mp3 and public/audio/score.mp3.
"""
import json, os, sys, urllib.request

SFX = {
    "crt": ("Old CRT computer monitor powering on: a soft thump, static crackle and a high-pitched whine", 1.5),
    "dial": ("Touch-tone telephone dialing a ten digit number quickly, clean DTMF beeps", 1.8),
    "modem": ("1980s acoustic coupler modem connecting: handshake screech, carrier tone and static", 2.2),
    "keys": ("Vintage teletype computer terminal printing text quickly, steady fast mechanical clicking", 4.0),
    "boom": ("Huge cinematic movie trailer impact boom with deep sub bass and long dark tail", 3.0),
    "klaxon": ("Military bunker alarm klaxon, DEFCON warning, two long blasts with echo", 3.0),
    "chirps": ("1980s mainframe computer processing data: rapid electronic beeps and chirps", 3.0),
    "riser": ("Cinematic tension riser building to a sharp cut, rising synth and noise", 2.5),
    "cut": ("Abrupt electrical power cut, a low thud then dead silence", 1.0),
}

SCORE = {
    "positive_global_styles": ["1983 analog synthesizer score", "cold war techno-thriller", "movie trailer",
                                "Moog bass", "instrumental"],
    "negative_global_styles": ["vocals", "lyrics", "modern EDM", "guitar", "happy"],
    "sections": [
        {"section_name": "Dial in", "duration_ms": 7600, "lines": [],
         "positive_local_styles": ["eerie low drone", "sparse", "quiet ticking pulse", "suspense"],
         "negative_local_styles": ["drums", "loud"]},
        {"section_name": "The games", "duration_ms": 4400, "lines": [],
         "positive_local_styles": ["driving sequenced synth arpeggio", "pulsing bass", "building energy"],
         "negative_local_styles": ["melody"]},
        {"section_name": "The war", "duration_ms": 8200, "lines": [],
         "positive_local_styles": ["massive", "intense", "pounding toms", "alarm-like brass synth stabs",
                                   "relentless build to a peak"],
         "negative_local_styles": ["calm"]},
        {"section_name": "Silence", "duration_ms": 4600, "lines": [],
         "positive_local_styles": ["almost silent", "one soft sustained synth pad", "contemplative"],
         "negative_local_styles": ["drums", "bass", "loud"]},
        {"section_name": "Title", "duration_ms": 5200, "lines": [],
         "positive_local_styles": ["ominous heroic synth theme", "slow swell", "final resolving chord"],
         "negative_local_styles": ["fast"]},
    ],
}

def post(path, payload, out):
    req = urllib.request.Request("https://api.elevenlabs.io" + path, data=json.dumps(payload).encode(),
                                 headers={"xi-api-key": os.environ["ELEVENLABS_API_KEY"],
                                          "Content-Type": "application/json"})
    with urllib.request.urlopen(req) as r, open(out, "wb") as f:
        f.write(r.read())
    print(out)

def main():
    for name in sys.argv[1:] or [*SFX, "score"]:
        if name == "score":
            post("/v1/music", {"composition_plan": SCORE}, "public/audio/score.mp3")
        else:
            text, secs = SFX[name]
            post("/v1/sound-generation", {"text": text, "duration_seconds": secs, "prompt_influence": 0.6},
                 f"public/audio/sfx-{name}.mp3")

if __name__ == "__main__":
    main()
