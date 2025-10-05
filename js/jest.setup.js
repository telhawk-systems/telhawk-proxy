// Jest setup file to provide Web APIs that may not be available in the test environment
const { TextEncoder, TextDecoder } = require('util');

// Polyfill TextEncoder and TextDecoder for the test environment
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;

// Ensure crypto is available for mocking in tests
if (typeof global.crypto === 'undefined') {
  global.crypto = {};
}
