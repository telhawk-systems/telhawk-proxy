import resolve from "@rollup/plugin-node-resolve";
import commonjs from "@rollup/plugin-commonjs";
import typescript from "@rollup/plugin-typescript";

/** @type {import('rollup').RollupOptions} */
export default {
  input: "src/index.ts",
  output: [
    {
      file: "dist/pixel.esm.js",
      format: "esm",
      sourcemap: true
    },
    {
      file: "dist/pixel.umd.js",
      format: "umd",
      name: "GoTrackPixel",
      sourcemap: true
    }
  ],
  plugins: [
    resolve({ browser: true, preferBuiltins: false }),
    commonjs(),
    typescript({
      tsconfig: "./tsconfig.json",
      declaration: true,
      declarationDir: "dist/types",
      compilerOptions: {
        outDir: undefined
      }
    })
  ]
};