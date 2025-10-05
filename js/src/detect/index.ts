import { runRegistry } from "./registry";
import { pluginsDetector } from "./plugins";
import { uaDetector } from "./userAgent";
import { permissionsDetector } from "./permissions";
import { webglDetector } from "./webgl";
import { webdriverDetector } from "./webdriver";
import { functionToStringDetector } from "./functionToString";
import { audioContextDetector } from "./audioContext";
import { apiMatrixDetector } from "./apiMatrix";
import { environmentInconsistencyDetector } from "./environmentInconsistency";
import { jsEngineFingerprintDetector } from "./jsEngineFingerprint";
import type { Detector } from "./types";

const detectors: Detector[] = [
  pluginsDetector, 
  uaDetector, 
  permissionsDetector, 
  webglDetector,
  webdriverDetector,
  functionToStringDetector,
  audioContextDetector,
  apiMatrixDetector,
  environmentInconsistencyDetector,
  jsEngineFingerprintDetector
];

export const runDetectors = () => runRegistry(detectors);