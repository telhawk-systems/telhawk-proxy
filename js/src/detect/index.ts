import { runRegistry } from "./registry";
import { pluginsDetector } from "./plugins";
import { uaDetector } from "./userAgent";
import { permissionsDetector } from "./permissions";
import { webglDetector } from "./webgl";
import type { Detector } from "./types";

const detectors: Detector[] = [pluginsDetector, uaDetector, permissionsDetector, webglDetector];

export const runDetectors = () => runRegistry(detectors);