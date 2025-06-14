package resource

import (
	"bytes"
	"encoding/base64"
	"io"
	"testing"

	"github.com/pulumi/pulumi-go-provider/infer/types"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/archive"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/asset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/tests/must"
)

func TestFile_buildFileCopyPlan(t *testing.T) {
	t.Parallel()

	b64 := func(s string) io.ReadCloser {
		b, err := base64.StdEncoding.DecodeString(s)
		require.NoError(t, err)
		return io.NopCloser(bytes.NewReader(b))
	}

	tests := map[string]struct {
		inputs FileArgs
		expect FileCopyPlan
	}{
		"inline content": {
			inputs: FileArgs{
				Content: ptr.Of("foo"),
			},
			expect: FileCopyPlan{
				// Strategy: FileCopyPlanInlineContent,
				Strategy: FileCopyPlanStringAsset,
				Reader:   b64("Zm9v"),
			},
		},

		"remote source": {
			inputs: FileArgs{
				RemoteSource: ptr.Of("/tmp/foo"),
			},
			expect: FileCopyPlan{
				Strategy: FileCopyPlanRemoteSource,
			},
		},

		"source file asset": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Asset: must.Must1(asset.FromPath("./testdata/file/testfile")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanFileAsset,
				Unarchive: false,
				Reader:    b64("aHR0cHM6Ly93d3cueW91dHViZS5jb20vd2F0Y2g/dj1uUzhFeXdYWWxTYwo="),
			},
		},

		"source string asset": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Asset: must.Must1(asset.FromText("foo")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanStringAsset,
				Unarchive: false,
				Reader:    b64("Zm9v"),
			},
		},

		"source remote asset": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Asset: must.Must1(asset.FromURI("https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testfile")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanRemoteAsset,
				Unarchive: false,
				Reader:    b64("aHR0cHM6Ly93d3cueW91dHViZS5jb20vd2F0Y2g/dj1uUzhFeXdYWWxTYwo="),
			},
		},

		"source dir": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Archive: must.Must1(archive.FromPath("./testdata/file/testdir")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanFileArchive,
				Unarchive: true,
				Reader:    b64("H4sIAAAAAAAA/+xWvY7dRg91rac43TYXwvX6D3Dj4vuQZIvEhQO4pjSUNLmjoTCkVlGXh8gT5kkCju7GPylSGHYQYNncK82Qc8hzzkALqW5Sgrb2qz35OnE+n88vz+f6e/777/n86sWH//7+6dNXL188wfkr4fkkVjUqT85ffNbnzf1HYpNyQaLFZEGSMebXzapcMs38GkqLJvqlcZHIVsJrqCQqW8xBn94+a/5t8I/xxaFrDrR/RfP/s/9vX94++9z/t8+fP/r/W8TbzAi0n0CDcUGIOXM5YZtiYsw7dlnzyAUa1dcpB9xh48JIvhLzCOpkNcSMH0vbfC/LxAXvRcKNYqcSTtgYuogZBxCGxGFMnve/icosecf/5Z5BKUEyoilkyy1+nsiajrodXSwBGyks5v2EaAgx5BsD33PGRA/JQ00emGzioqeKNRp6WZNvbzTmscX7ibMjGmSt67WgV6dUmML+ACKRGrri1U5+iCMeKCXOiNkE1OhU1g5//PY7liIddWkHdZSDZA7o9lplocLZtMV7RuA+Bg4wQbfG5MPIrIZBCqI1JY6TwbFzhW4T5xY/yMb3zohNMV8cREf9xR/JsMXspDj6Na+6Uko7eknh1GzRJgyRS8+o9zXIkOsRMXvtKzdZzAHNnC3KsTBT3rFIVMmyKrp11MYBbY65Y1LThxr3sY852t62Le6OMfaJqRzw4uCDTjzUtj4w33yCIBrURAKyoJ8o9+xU6lruYx36mi0m6FIqfe/khDvoOo6sLqiNG6MLe5WYNQY+YUnU1xfXfnTiNDgh/uCzkO0QB+mF65xBYU2mMGkG0skHQehp5Cs3n9JXJ+8qiobCI0Xn28lWK5xHm8BZ1nHycrq4qOrqFvOo1QybKxKFE5NWoE7poaoDYgquf4YVGmPvIjLXWaMys+tgvILgm4Ds8kAvtfnCoQqyusW32BS1GuhGMZAxJgoPSm865ozAxmWuPSRxefEghY/EWVwXB7mKwOoOrMU/yup2yFqaYy+V3e+OucVPPuU7+KdD7WqRjYsz208Se3bR7bK6c1t8R71T6ZL1cxuNtpLr0S8isoeUY26eNdOF3+Cdxf6C69SkxDFmSk5/Pvi9uqypLqvVVQZD5spfbZI/u4UGTukN3pa/HPqRCmrRZmA+ro3xkGc8xL3aLGroqTCGInM976rzjcpsk/dOmGTmN7gD08gl7Q1tFM178rtVNy7t41fVYzzGY3yb+DMAAP//KsZIzQAQAAA="),
			},
		},

		"source tar.gz archive": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Archive: must.Must1(archive.FromPath("./testdata/file/testdir.tar.gz")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanFileArchive,
				Unarchive: true,
				Reader:    b64("H4sIAAAAAAAA/+xWvY7cRgx2raf4umsWm/WdYwNuXCRIckXiwgFcUxpKmuxoKAypU9TlIfKEeZKAo93EP0WKOAcEODZaaYacj/w+ctZYLcTylS450Ha0X+3ZF7fT6XR6eTrV5+nz5+3L27vr7/3789tXt7fPcPryUD63RY3Ks9O/PuvT5P4n9jYzAm0HUG9cEGLOXA5Yx5gY04ZNljxwgUb1dcoB91i5MJKvxDyAWlkMMePHcmy+l3nkgvci4UaxUQkHrAydxYwDCH3iMCT3+2akMkne8K08MCglSEY0haz5iJ9HsqaldkMbS8BKCot5OyAaQgz5xsAPnDHS1bmvzj2TjVz0ULFGQydL8u2Nxjwc8X7k7Ih6Wep6DejRKRWmsF1BJFJDWzzawQ9xxD2lxBkxm4AaHcvS4o/ffsdcpKU2baCWcpDMAe1Wo8xUOJse8Z4RuIuBA0zQLjF5MTKroZeCaE2Jw2hw7Fyh28j5iB9k5QdnxMaYzw6ipe7sr2RYY3ZSHP2SF10opQ2dpHBo1mgj+silY98WFGTI9YiYPfaFmyzmgCbOFmVfmChvmCWqZFkU7TJo44BWx9wyqek1xkPsYo62HY9H3O9l7BJT2eHF3guduK9p/c188xGCaFATCciCbqTcsVOpS3mItehLtpigc6n0vZMD7qHLMLC6oFZujM7sUWLWGPiAOVFXP1zy0ZFT74T4i9dC1l0cpGeudQaFJZnCpOlJRy8EoaOBL9x8TF+tvKsoGgoPFJ1vJ1utcB5sBGdZhtHD6eyiqqtrzIPWZlhdkSicmLQCdUp3Ve0QU3D9M6zQEDsXkbnOGpWJXQfDBQTfBGSXBzqpyRcOVZC1W3yLjVFrA90oejLGSOGq9KZlzghsXKaaQxKXF/dSeHecxHWxk6sIrN6BNfgHXu0GWUqz76Wy+eyYjvjJq3yPmVRrVrOsXJzZbpTYsYtuk8U794jvqHMqXbJ+bqPRFnI9+iAiu7rsdXOvic78Bu8sdmdcqiYlDjFTcvrzzu+ly5raZTW6Sm/IXPmrSfInU6jnlN7gbfmrQz9QQQ3a9Mz72Bh2ecZd3ItNooaOCqMvMtXzLjpfqUw2eu6EUSZ+g3swDVzS1tBK0Twnn626cjk2jzr/r/e/87RKCfof/AX4h/v/dHr19Sf3/92LF3dP9/9j2CrljESzyYwkQ8yvm0W5ZJr4NZRmTfRL4+KQtYTXUElU6n3y/PbucZX6ZE/2ZE/2ZF/S/gwAAP//UDnFrwAQAAA="),
			},
		},

		"source zip archive": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Archive: must.Must1(archive.FromPath("./testdata/file/testdir.zip")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanFileArchive,
				Unarchive: true,
				Reader:    b64("H4sIAAAAAAAA/+xWva7cRg91rac43W0W+63v9WcDblwkSHKLxIUDuKY0lDTZ0VAYUldRl4fIE+ZJAo72xj8JkMKOgwDLZleaIeeQ55yBjNVCLP+bSXWVEvRoP9uTzxun0+n0/HSqv6c//55OL/7/7r+/f3r37NndE5w+M46/jEWNypPTJ5/1cXP/kVilnJFoNpmRZIj5ZbMol0wTv4TSrIl+alwcspbwEiqJyhpz0Ke3d82/Df4anxyP/tclB9r+AfP/vf9vn9/efeT/2xe3t1f/f4l4nRmBtgOoNy4IMWcuB6xjTIxpwyZLHrhAo/o65YB7rFwYyVdiHkCtLIaY8X05Nt/KPHLBW5Fwo9iohANWhs5ixgGEPnEYkud9NVKZJG/4Wh4YlBIkI5pC1nzEjyNZ01K7oY0lYCWFxbwdEA0hhnxj4AfOGOkxua/JPZONXPRQsUZDJ0vy7Y3GPBzxduTsiHpZ6not6NUpFaawPYJIpIa2eLWDH+KIe0qJM2I2ATU6lqXFb7/8irlIS23aQC3lIJkD2q1WmalwNj3iLSNwFwMHmKBdYvJhZFZDLwXRmhKH0eDYuUK3kfMR38nKD86IjTGfHURL3dkfybDG7KQ4+iUvulBKGzpJ4dCs0Ub0kUvHqPc1yJDrETF77Qs3WcwBTZwtyr4wUd4wS1TJsijaZdDGAa2OuWVS08caD7GLOdp2PB5xv4+xS0xlhxd7H3Tivrb1jvnmAwTRoCYSkAXdSLljp1KX8hDr0JdsMUHnUul7IwfcQ5dhYHVBrdwYndmrxKwx8AFzoq6+uPSjI6feCfEHn4WsuzhIz1znDApLMoVJ05OOPghCRwNfuPmQvjp5V1E0FB4oOt9OtlrhPNgIzrIMo5fT2UVVV9eYB61mWF2RKJyYtAJ1SndV7RBTcP0zrNAQOxeRuc4alYldB8MFBN8EZJcHOqnNFw5VkNUtvsXGqNVAN4qejDFSeFR60zJnBDYuU+0hicuLeym8J07iutjJVQRWd2At/l5Wu0GW0ux7qWx+d0xH/OBTvod/OtSuZlm5OLPdKLFjF90mizv3iG+ocypdsn5uo9EWcj36RUT2mLLPzbMmOvMrvLHYnXGZmpQ4xEzJ6c87vxeXNdVltbpKb8hc+atN8ke3UM8pvcLr8odD31NBLdr0zPu1MezyjLu4F5tEDR0VRl9kqudddL5SmWz03gmjTPwK92AauKStoZWieU9+t+rK5Xj9qrrGNa7xZeL3AAAA///JUKuOABAAAA=="),
			},
		},

		"source remote tar.gz archive": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Archive: must.Must1(archive.FromURI("https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testdir.tar.gz")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanRemoteArchive,
				Unarchive: true,
				Reader:    b64("H4sIAAAAAAAA/+xWvY7cRgx2raf4umsWm/WdYwNuXCRIckXiwgFcUxpKmuxoKAypU9TlIfKEeZKAo93EP0WKOAcEODZaaYacj/w+ctZYLcTylS450Ha0X+3ZF7fT6XR6eTrV5+nz5+3L27vr7/3789tXt7fPcPryUD63RY3Ks9O/PuvT5P4n9jYzAm0HUG9cEGLOXA5Yx5gY04ZNljxwgUb1dcoB91i5MJKvxDyAWlkMMePHcmy+l3nkgvci4UaxUQkHrAydxYwDCH3iMCT3+2akMkne8K08MCglSEY0haz5iJ9HsqaldkMbS8BKCot5OyAaQgz5xsAPnDHS1bmvzj2TjVz0ULFGQydL8u2Nxjwc8X7k7Ih6Wep6DejRKRWmsF1BJFJDWzzawQ9xxD2lxBkxm4AaHcvS4o/ffsdcpKU2baCWcpDMAe1Wo8xUOJse8Z4RuIuBA0zQLjF5MTKroZeCaE2Jw2hw7Fyh28j5iB9k5QdnxMaYzw6ipe7sr2RYY3ZSHP2SF10opQ2dpHBo1mgj+silY98WFGTI9YiYPfaFmyzmgCbOFmVfmChvmCWqZFkU7TJo44BWx9wyqek1xkPsYo62HY9H3O9l7BJT2eHF3guduK9p/c188xGCaFATCciCbqTcsVOpS3mItehLtpigc6n0vZMD7qHLMLC6oFZujM7sUWLWGPiAOVFXP1zy0ZFT74T4i9dC1l0cpGeudQaFJZnCpOlJRy8EoaOBL9x8TF+tvKsoGgoPFJ1vJ1utcB5sBGdZhtHD6eyiqqtrzIPWZlhdkSicmLQCdUp3Ve0QU3D9M6zQEDsXkbnOGpWJXQfDBQTfBGSXBzqpyRcOVZC1W3yLjVFrA90oejLGSOGq9KZlzghsXKaaQxKXF/dSeHecxHWxk6sIrN6BNfgHXu0GWUqz76Wy+eyYjvjJq3yPmVRrVrOsXJzZbpTYsYtuk8U794jvqHMqXbJ+bqPRFnI9+iAiu7rsdXOvic78Bu8sdmdcqiYlDjFTcvrzzu+ly5raZTW6Sm/IXPmrSfInU6jnlN7gbfmrQz9QQQ3a9Mz72Bh2ecZd3ItNooaOCqMvMtXzLjpfqUw2eu6EUSZ+g3swDVzS1tBK0Twnn626cjk2jzr/r/e/87RKCfof/AX4h/v/dHr19Sf3/92LF3dP9/9j2CrljESzyYwkQ8yvm0W5ZJr4NZRmTfRL4+KQtYTXUElU6n3y/PbucZX6ZE/2ZE/2ZF/S/gwAAP//UDnFrwAQAAA="),
			},
		},

		"source remote zip archive": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Archive: must.Must1(archive.FromURI("https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testdir.zip")),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanRemoteArchive,
				Unarchive: true,
				Reader:    b64("H4sIAAAAAAAA/+xWva7cRg91rac43W0W+63v9WcDblwkSHKLxIUDuKY0lDTZ0VAYUldRl4fIE+ZJAo72xj8JkMKOgwDLZleaIeeQ55yBjNVCLP+bSXWVEvRoP9uTzxun0+n0/HSqv6c//55OL/7/7r+/f3r37NndE5w+M46/jEWNypPTJ5/1cXP/kVilnJFoNpmRZIj5ZbMol0wTv4TSrIl+alwcspbwEiqJyhpz0Ke3d82/Df4anxyP/tclB9r+AfP/vf9vn9/efeT/2xe3t1f/f4l4nRmBtgOoNy4IMWcuB6xjTIxpwyZLHrhAo/o65YB7rFwYyVdiHkCtLIaY8X05Nt/KPHLBW5Fwo9iohANWhs5ixgGEPnEYkud9NVKZJG/4Wh4YlBIkI5pC1nzEjyNZ01K7oY0lYCWFxbwdEA0hhnxj4AfOGOkxua/JPZONXPRQsUZDJ0vy7Y3GPBzxduTsiHpZ6not6NUpFaawPYJIpIa2eLWDH+KIe0qJM2I2ATU6lqXFb7/8irlIS23aQC3lIJkD2q1WmalwNj3iLSNwFwMHmKBdYvJhZFZDLwXRmhKH0eDYuUK3kfMR38nKD86IjTGfHURL3dkfybDG7KQ4+iUvulBKGzpJ4dCs0Ub0kUvHqPc1yJDrETF77Qs3WcwBTZwtyr4wUd4wS1TJsijaZdDGAa2OuWVS08caD7GLOdp2PB5xv4+xS0xlhxd7H3Tivrb1jvnmAwTRoCYSkAXdSLljp1KX8hDr0JdsMUHnUul7IwfcQ5dhYHVBrdwYndmrxKwx8AFzoq6+uPSjI6feCfEHn4WsuzhIz1znDApLMoVJ05OOPghCRwNfuPmQvjp5V1E0FB4oOt9OtlrhPNgIzrIMo5fT2UVVV9eYB61mWF2RKJyYtAJ1SndV7RBTcP0zrNAQOxeRuc4alYldB8MFBN8EZJcHOqnNFw5VkNUtvsXGqNVAN4qejDFSeFR60zJnBDYuU+0hicuLeym8J07iutjJVQRWd2At/l5Wu0GW0ux7qWx+d0xH/OBTvod/OtSuZlm5OLPdKLFjF90mizv3iG+ocypdsn5uo9EWcj36RUT2mLLPzbMmOvMrvLHYnXGZmpQ4xEzJ6c87vxeXNdVltbpKb8hc+atN8ke3UM8pvcLr8odD31NBLdr0zPu1MezyjLu4F5tEDR0VRl9kqudddL5SmWz03gmjTPwK92AauKStoZWieU9+t+rK5Xj9qrrGNa7xZeL3AAAA///JUKuOABAAAA=="),
			},
		},

		"source asset archive": {
			inputs: FileArgs{
				Source: &types.AssetOrArchive{
					Archive: must.Must1(archive.FromAssets(map[string]any{
						"testfile": must.Must1(asset.FromPath("./testdata/file/testfile")),
					})),
				},
			},
			expect: FileCopyPlan{
				Strategy:  FileCopyPlanAssetArchive,
				Unarchive: true,
				Reader:    b64("H4sIAAAAAAAA/ypJLS5Jy8xJZaAhMDAwMDAzMADTBpi0gYGpCYINEjc0MDc2Y1AwoKWjYKC0uCSxiMGAYrvQPTdEQEZJSUGxlb5+eXm5XmV+aUlpUqpecn6ufnliSXKGfZltXrCFa2V5RGROcDLXQLt1FIyCUTAKRgH1ACAAAP//EON/MgAIAAA="),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := File{}

			got, err := r.buildFileCopyPlan(tc.inputs)
			require.NoError(t, err)

			assert.Equal(t, tc.expect.Strategy, got.Strategy)
			assert.Equal(t, tc.expect.Unarchive, got.Unarchive)

			if tc.expect.Reader == nil {
				assert.Nil(t, got.Reader)
			} else {
				expectData, err := io.ReadAll(tc.expect.Reader)
				require.NoError(t, err)
				require.NotNil(t, got.Reader)
				gotData, err := io.ReadAll(got.Reader)
				require.NoError(t, err)
				assert.Equal(t, expectData, gotData)
			}
		})
	}
}

func TestFile_argsToFileTaskParameters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputs FileArgs
		expect ansible.FileParameters
	}{
		"minimal": {
			inputs: FileArgs{
				Path: "/tmp/foo",
			},
			expect: ansible.FileParameters{
				Path: "/tmp/foo",
			},
		},

		"full": {
			inputs: FileArgs{
				AccessTime:             ptr.Of("197001010000.00"),
				AccessTimeFormat:       ptr.Of("%Y%m%d%H%M.%S"),
				Attributes:             ptr.Of("+d"),
				Backup:                 ptr.Of(true),
				Ensure:                 ptr.Of(FileEnsureDirectory),
				Follow:                 ptr.Of(true),
				Group:                  ptr.Of("root"),
				Mode:                   ptr.Of("u+rwx"),
				ModificationTime:       ptr.Of("197001010000.00"),
				ModificationTimeFormat: ptr.Of("%Y%m%d%H%M.%S"),
				Owner:                  ptr.Of("root"),
				Path:                   "/tmp/foo",
				Recurse:                ptr.Of(true),
				RemoteSource:           ptr.Of("/tmp/bar"),
				Selevel:                ptr.Of("_default"),
				Serole:                 ptr.Of("_default"),
				Setype:                 ptr.Of("_default"),
				Seuser:                 ptr.Of("_default"),
				UnsafeWrites:           ptr.Of(true),
			},
			expect: ansible.FileParameters{
				AccessTime:             ptr.Of("197001010000.00"),
				AccessTimeFormat:       ptr.Of("%Y%m%d%H%M.%S"),
				Attributes:             ptr.Of("+d"),
				Follow:                 ptr.Of(true),
				Group:                  ptr.Of("root"),
				Mode:                   ptr.ToAny(ptr.Of("u+rwx")),
				ModificationTime:       ptr.Of("197001010000.00"),
				ModificationTimeFormat: ptr.Of("%Y%m%d%H%M.%S"),
				Owner:                  ptr.Of("root"),
				Path:                   "/tmp/foo",
				Recurse:                ptr.Of(true),
				Selevel:                ptr.Of("_default"),
				Serole:                 ptr.Of("_default"),
				Setype:                 ptr.Of("_default"),
				Seuser:                 ptr.Of("_default"),
				Src:                    ptr.Of("/tmp/bar"),
				State:                  ptr.Of(ansible.FileStateDirectory),
				UnsafeWrites:           ptr.Of(true),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := File{}

			got, err := r.argsToFileTaskParameters(tc.inputs)
			assert.NoError(t, err)
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestFile_argsToCopyTaskParameters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputs FileArgs
		expect ansible.CopyParameters
	}{
		"minimal": {
			inputs: FileArgs{
				Path: "/tmp/foo",
			},
			expect: ansible.CopyParameters{
				Dest:      "/tmp/foo",
				RemoteSrc: ptr.Of(true),
			},
		},

		"full": {
			inputs: FileArgs{
				AccessTime:             ptr.Of("197001010000.00"),
				AccessTimeFormat:       ptr.Of("%Y%m%d%H%M.%S"),
				Attributes:             ptr.Of("+d"),
				Backup:                 ptr.Of(true),
				Ensure:                 ptr.Of(FileEnsureDirectory),
				Follow:                 ptr.Of(true),
				Force:                  ptr.Of(true),
				Group:                  ptr.Of("root"),
				Mode:                   ptr.Of("u+rwx"),
				ModificationTime:       ptr.Of("197001010000.00"),
				ModificationTimeFormat: ptr.Of("%Y%m%d%H%M.%S"),
				Owner:                  ptr.Of("root"),
				Path:                   "/tmp/foo",
				Recurse:                ptr.Of(true),
				RemoteSource:           ptr.Of("/tmp/bar"),
				Selevel:                ptr.Of("_default"),
				Serole:                 ptr.Of("_default"),
				Setype:                 ptr.Of("_default"),
				Seuser:                 ptr.Of("_default"),
				UnsafeWrites:           ptr.Of(true),
				Validate:               ptr.Of("true"),
			},
			expect: ansible.CopyParameters{
				Attributes:   ptr.Of("+d"),
				Backup:       ptr.Of(true),
				Dest:         "/tmp/foo",
				Follow:       ptr.Of(true),
				Force:        ptr.Of(true),
				Group:        ptr.Of("root"),
				Mode:         ptr.ToAny(ptr.Of("u+rwx")),
				Owner:        ptr.Of("root"),
				RemoteSrc:    ptr.Of(true),
				Selevel:      ptr.Of("_default"),
				Serole:       ptr.Of("_default"),
				Setype:       ptr.Of("_default"),
				Seuser:       ptr.Of("_default"),
				UnsafeWrites: ptr.Of(true),
				Validate:     ptr.Of("true"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := File{}

			got, err := r.argsToCopyTaskParameters(tc.inputs)
			assert.NoError(t, err)
			assert.Equal(t, tc.expect, got)
		})
	}
}
