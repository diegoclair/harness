import { loadKit } from "./kit.mjs";
import { mountTable, registerMounts } from "./mounts.mjs";

// Loaded with --import, so a kit script the engine runs reaches the engine, the kit and its mounts by the page's paths.
registerMounts(mountTable(loadKit(process.env.FILM_KIT)));
