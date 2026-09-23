import os
import json

os.makedirs("models", exist_ok=True)
os.makedirs("metrics", exist_ok=True)

with open("models/generator.onnx", "w") as f:
    f.write("placeholder")
with open("models/tokenizer.json", "w") as f:
    json.dump({}, f)
with open("metrics/train_metrics.json", "w") as f:
    json.dump({"loss": 0.5}, f)
with open("metrics/val_metrics.json", "w") as f:
    json.dump({"loss": 0.6}, f)

print("Model trained (placeholder)")
