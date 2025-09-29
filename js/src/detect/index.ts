import { runRegistry } from "./registry";
import { pluginsDetector } from "./plugins";
import { uaDetector } from "./userAgent";
import { permissionsDetector } from "./permissions";
import { webglDetector } from "./webgl";
import { webdriverDetector } from "./webdriver";
import { functionToStringDetector } from "./functionToString";
import type { Detector } from "./types";

const detectors: Detector[] = [
  pluginsDetector, 
  uaDetector, 
  permissionsDetector, 
  webglDetector,
  webdriverDetector,
  functionToStringDetector
];

export const runDetectors = () => runRegistry(detectors);