import type { Detector } from "./types";

export const pluginsDetector: Detector = {
  id: "plugins",
  run: () => {
    let plugins: string[] = [];
    let mimeTypes: string[] = [];
    let pluginsLen = 0;
    let mimeLen = 0;

    try {
      pluginsLen = navigator.plugins?.length ?? 0;
      mimeLen = navigator.mimeTypes?.length ?? 0;
      
      // Collect plugin names for security analysis
      if (navigator.plugins) {
        for (let i = 0; i < navigator.plugins.length; i++) {
          const plugin = navigator.plugins[i];
          if (plugin?.name) {
            plugins.push(plugin.name);
          }
        }
      }
      
      // Collect mime types for security analysis
      if (navigator.mimeTypes) {
        for (let i = 0; i < navigator.mimeTypes.length; i++) {
          const mime = navigator.mimeTypes[i];
          if (mime?.type) {
            mimeTypes.push(mime.type);
          }
        }
      }
    } catch {}

    const suspicious = pluginsLen === 0 && mimeLen === 0;
    const score = suspicious ? 2 : (pluginsLen < 3 ? 1 : 0); // More aggressive scoring
    
    return { 
      id: "plugins", 
      score, 
      details: { 
        pluginsLen, 
        mimeLen, 
        plugins: plugins.slice(0, 20), // Limit to first 20 to avoid huge payloads
        mimeTypes: mimeTypes.slice(0, 20),
        suspicious
      } 
    };
  }
};