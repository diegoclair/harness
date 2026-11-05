import os, subprocess, sys, tempfile, unittest
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import numpy as np
import footage


class Regions(unittest.TestCase):
    def test_a_box_moves_between_its_keys(self):
        self.assertEqual(footage.key_at({0: (0, 0, 10, 10), 2: (10, 0, 20, 10)}, 1), (5.0, 0.0, 15.0, 10.0))

    def test_a_box_holds_past_its_last_key(self):
        self.assertEqual(footage.key_at({0: (0, 0, 10, 10)}, 9), (0, 0, 10, 10))

    def test_the_whole_box_is_covered_up_to_its_edge(self):
        m = footage.mask_at([{0: (20, 20, 60, 60)}], 0, (80, 80))
        self.assertTrue((m[20:60, 20:60] == 1).all())

    def test_only_the_region_is_blurred(self):
        f = np.random.default_rng(1).uniform(0, 255, (80, 80, 3)).astype(np.float32)
        out = footage.blur(f, [{0: (0, 0, 20, 20)}], 0)
        self.assertLess(out[5:15, 5:15].std(), f[5:15, 5:15].std() / 2)
        self.assertTrue((out[60:, 60:] == f[60:, 60:]).all())


class Guards(unittest.TestCase):
    def test_raw_footage_inside_a_repository_is_refused(self):
        with self.assertRaises(SystemExit):
            footage.refuse_private_leak(os.path.abspath(__file__), tempfile.mkdtemp())

    def test_an_output_git_would_track_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = os.path.join(tmp, "repo"); raw = os.path.join(tmp, "raw.mp4")
            os.makedirs(repo); open(raw, "w").close()
            subprocess.run(["git", "init", "-q", repo], check=True)
            with self.assertRaises(SystemExit):
                footage.refuse_private_leak(raw, os.path.join(repo, "clips", "x"))


class Shot(unittest.TestCase):
    def test_a_shot_comes_out_and_is_sampled_every_half_second(self):
        with tempfile.TemporaryDirectory() as tmp:
            subprocess.run(["ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=576x1024:rate=30:duration=3", os.path.join(tmp, "raw.mp4")], check=True)
            clips = os.path.join(tmp, "clips")
            footage.render_shot("a", dict(src="raw.mp4", a=0.0, b=2.0, dur=1.0, regions=[{0: (0, 0, 576, 200)}]), tmp, clips)
            self.assertEqual(len(os.listdir(os.path.join(clips, "a"))), 30)
            self.assertEqual(len(footage.sample(clips, os.path.join(tmp, "sheets"))), 1)


if __name__ == "__main__":
    unittest.main()
