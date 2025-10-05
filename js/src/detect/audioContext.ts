import type { Detector } from "./types";

export const audioContextDetector: Detector = {
  id: "audio_context",
  run: () => {
    let sampleRate = 0;
    let maxChannelCount = 0;
    let numberOfInputs = 0;
    let numberOfOutputs = 0;
    let state = "";
    let baseLatency = 0;
    let outputLatency = 0;
    let available = false;

    try {
      const AudioContextClass = (window as any).AudioContext || (window as any).webkitAudioContext;
      if (!AudioContextClass) {
        return createResult();
      }

      const ctx = new AudioContextClass();
      available = true;
      
      // Collect audio context properties
      sampleRate = ctx.sampleRate || 0;
      maxChannelCount = ctx.destination?.maxChannelCount || 0;
      numberOfInputs = ctx.destination?.numberOfInputs || 0;
      numberOfOutputs = ctx.destination?.numberOfOutputs || 0;
      state = ctx.state || "";
      baseLatency = ctx.baseLatency || 0;
      outputLatency = ctx.outputLatency || 0;

      // Clean up
      ctx.close?.();

    } catch (e) {
      // Error details captured in available flag
    }

    function createResult() {
      return { 
        id: "audio_context", 
        score: 0, // No scoring, just raw data
        details: { 
          available,
          sampleRate,
          maxChannelCount,
          numberOfInputs,
          numberOfOutputs,
          state,
          baseLatency,
          outputLatency
        } 
      };
    }

    return createResult();
  }
};