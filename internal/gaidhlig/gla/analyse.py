#!/usr/bin/env python3
import sys
import os
import ufal.udpipe as udpipe

model_path = os.path.join(os.path.dirname(__file__), "scottish_gaelic.udpipe")
model = udpipe.Model.load(model_path)
if not model:
    print("Error: could not load model", file=sys.stderr)
    sys.exit(1)

pipeline = udpipe.Pipeline(
    model,
    "tokenize",
    udpipe.Pipeline.DEFAULT,
    udpipe.Pipeline.DEFAULT,
    "conllu"
)

text = sys.stdin.read()
print(pipeline.process(text))
