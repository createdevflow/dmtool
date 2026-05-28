from fastapi import FastAPI
import uvicorn

app = FastAPI(title="AI Engine for DMTool")

@app.post("/ai/analyze")
async def analyze_data():
    return {
        "issue": "Low engagement",
        "reason": "Weak content hook",
        "solution": "Use curiosity-based hooks"
    }

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
