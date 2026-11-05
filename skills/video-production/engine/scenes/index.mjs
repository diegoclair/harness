// The engine's own scenes carry no brand; a kit scene with the same name wins, which is the only override there is.
const ENGINE_SCENES = {};

export async function sceneLibrary(kitScenes) {
  return { ...ENGINE_SCENES, ...(await import(kitScenes)).SCENES };
}
