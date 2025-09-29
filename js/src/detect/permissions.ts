import type { Detector } from "./types";

export const permissionsDetector: Detector = {
  id: "permissions",
  run: async () => {
    const permissions: any = {};
    let suspicious = false;
    let available = false;

    try {
      if (typeof navigator !== "undefined" && (navigator as any).permissions?.query) {
        available = true;
        
        // Test multiple permissions for security analysis
        const permissionsToTest = [
          'camera', 'microphone', 'geolocation', 'notifications', 
          'persistent-storage', 'push', 'midi'
        ];
        
        for (const perm of permissionsToTest) {
          try {
            const result = await (navigator as any).permissions.query({ name: perm });
            permissions[perm] = result.state;
            
            // In headless/automation, permissions often behave strangely
            if (result.state === "prompt" && perm === "camera") {
              suspicious = true; // Camera should rarely be in prompt state
            }
          } catch (e) {
            permissions[perm] = `error: ${e}`;
          }
        }
        
        // Check for automation-specific permission behaviors
        if (Object.values(permissions).every(state => state === "denied")) {
          suspicious = true; // All permissions denied is suspicious
        }
      }
    } catch (e) {
      permissions.error = String(e);
    }

    return { 
      id: "permissions", 
      score: suspicious ? 1 : 0, 
      details: { 
        available,
        permissions,
        suspicious,
        permissionCount: Object.keys(permissions).length
      } 
    };
  }
};
