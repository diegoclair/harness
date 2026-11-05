// The one place a QA limit is set: a check that disagrees with a film changes here, never inside the check.
export const THRESHOLDS = {
  durationToleranceSec: 0.15,
  // Only a film that declares a target is held to one: an approved script sets its own length.
  targetToleranceSec: 3,
  loudnessLufs: -14,
  loudnessToleranceLu: 0.5,
  truePeakDbtp: -1,
  firstSpeechSec: 0.5,
  hookEndSec: 3,
  transcriptMaxMismatch: 0.1,
  lineMinCoverage: 0.5,
  // A caption follows speech, so its floor is what a phone reader catches, not the reading rule of static text.
  captionMinSec: 0.45,
  captionMaxWps: 6,
  stillMaxSec: 2.5,
  // Mean absolute luma difference, 0 to 255, on the small grey frames QA samples.
  stillDiff: 1.2,
  stillSampleFps: 10,
  frame0MinStd: 8,
  readingWps: 3,
  readingExtraSec: 1,
};
