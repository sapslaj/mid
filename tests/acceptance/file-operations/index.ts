import * as aws from "@pulumi/aws";
import { local, remote } from "@pulumi/command";
import * as pulumi from "@pulumi/pulumi";
import * as tls from "@pulumi/tls";
import * as mid from "@sapslaj/pulumi-mid";

const sg = new aws.ec2.SecurityGroup("file-operations", {
  egress: [
    {
      fromPort: 0,
      toPort: 0,
      protocol: "-1",
      cidrBlocks: ["0.0.0.0/0"],
      ipv6CidrBlocks: ["::/0"],
    },
  ],
  ingress: [
    // protip: don't actually allow inbound SSH and file-operations ports from
    // anywhere in real code unless you know what you are doing.
    {
      fromPort: 22,
      toPort: 22,
      protocol: "tcp",
      cidrBlocks: ["0.0.0.0/0"],
      ipv6CidrBlocks: ["::/0"],
    },
    {
      fromPort: 5432,
      toPort: 5432,
      protocol: "tcp",
      cidrBlocks: ["0.0.0.0/0"],
      ipv6CidrBlocks: ["::/0"],
    },
  ],
});

const privateKey = new tls.PrivateKey("file-operations", {
  algorithm: "ED25519",
});

const keypair = new aws.ec2.KeyPair("file-operations", {
  publicKey: privateKey.publicKeyOpenssh,
});

const role = new aws.iam.Role("file-operations", {
  assumeRolePolicy: aws.iam.getPolicyDocumentOutput({
    statements: [
      {
        actions: ["sts:AssumeRole"],
        principals: [
          {
            type: "Service",
            identifiers: ["ec2.amazonaws.com"],
          },
        ],
      },
    ],
  }).json,
  managedPolicyArns: [
    "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
  ],
});

const instanceProfile = new aws.iam.InstanceProfile("file-operations", {
  name: role.name,
  role: role.name,
});

const instance = new aws.ec2.Instance("file-operations", {
  ami: aws.ec2.getAmiOutput({
    mostRecent: true,
    owners: ["099720109477"],
    filters: [
      {
        name: "virtualization-type",
        values: ["hvm"],
      },
      {
        name: "name",
        values: ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*"],
      },
    ],
  }).id,
  iamInstanceProfile: instanceProfile.name,
  instanceType: "t4g.nano",
  vpcSecurityGroupIds: [sg.id],
  keyName: keypair.keyName,
}, {});

const connection = {
  host: instance.publicIp,
  user: "ubuntu",
  privateKey: privateKey.privateKeyOpenssh,
};

const provider = new mid.Provider("file-operations", {
  connection,
});

// needed for some remoteSource functions
const unzip = new mid.resource.Package("unzip", {
  name: "unzip",
}, {
  provider,
});

function testfileAssertion(test: string): string {
  return `
    set -eux
    test -f "/tmp/${test}"
    cat "/tmp/${test}"
    test "https://www.youtube.com/watch?v=nS8EywXYlSc" = "$(cat "/tmp/${test}")"
  `;
}

function testdirAssertion(test: string): string {
  return `
    set -eux
    ls -lah "/tmp/${test}" || true
    test -d "/tmp/${test}"
    test -f "/tmp/${test}/passwords.txt"
    test -f "/tmp/${test}/sunday.txt"
    grep -F "solarwinds123" "/tmp/${test}/passwords.txt"
    grep -F "Charmony" "/tmp/${test}/sunday.txt"
  `;
}

const emptyFile = new mid.resource.File("empty-file", {
  path: "/tmp/empty-file",
  ensure: "file",
}, {
  provider,
});

new remote.Command("empty-file", {
  connection,
  create: `
    set -eux
    test -f "/tmp/empty-file"
    cat "/tmp/empty-file"
    test -z "$(cat "/tmp/empty-file")"
  `,
}, {
  dependsOn: [emptyFile],
});

const emptyDirectory = new mid.resource.File("empty-directory", {
  path: "/tmp/empty-directory",
  ensure: "directory",
}, {
  provider,
});

new remote.Command("empty-directory", {
  connection,
  create: `
    set -eux
    test -d /tmp/empty-directory
    test -z "$(ls /tmp/empty-directory)"
  `,
}, {
  dependsOn: [emptyDirectory],
});

const inlineContent = new mid.resource.File("inline-content", {
  path: "/tmp/inline-content",
  content: "https://www.youtube.com/watch?v=nS8EywXYlSc\n",
}, {
  provider,
});

new remote.Command("inline-content", {
  connection,
  create: testfileAssertion("inline-content"),
}, {
  dependsOn: [inlineContent],
});

const localSourceStringAsset = new mid.resource.File("local-source-string-asset", {
  path: "/tmp/local-source-string-asset",
  source: new pulumi.asset.StringAsset("https://www.youtube.com/watch?v=nS8EywXYlSc\n"),
}, {
  provider,
});

new remote.Command("local-source-string-asset", {
  connection,
  create: testfileAssertion("local-source-string-asset"),
}, {
  dependsOn: [localSourceStringAsset],
});

const localSourceFileAsset = new mid.resource.File("local-source-file-asset", {
  path: "/tmp/local-source-file-asset",
  source: new pulumi.asset.FileAsset(__dirname + "/testdata/testfile"),
}, {
  provider,
});

new remote.Command("local-source-file-asset", {
  connection,
  create: testfileAssertion("local-source-file-asset"),
}, {
  dependsOn: [localSourceFileAsset],
});

const localSourceGeneratedFileAssetCommand = new local.Command("local-source-generated-file-asset", {
  create: "true",
  assetPaths: [
    "testdata/testfile",
  ],
});

const localSourceGeneratedFileAsset = new mid.resource.File("local-source-generated-file-asset", {
  path: "/tmp/local-source-generated-file-asset",
  source: localSourceGeneratedFileAssetCommand.assets.apply((assets) => assets!["testdata/testfile"]),
}, {
  provider,
});

new remote.Command("local-source-generated-file-asset", {
  connection,
  create: testfileAssertion("local-source-generated-file-asset"),
}, {
  dependsOn: [localSourceGeneratedFileAsset],
});

const localNetworkSourceAsset = new mid.resource.File("local-network-source-asset", {
  path: "/tmp/local-network-source-asset",
  source: new pulumi.asset.RemoteAsset("https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testfile"),
}, {
  provider,
});

new remote.Command("local-network-source-asset", {
  connection,
  create: testfileAssertion("local-network-source-asset"),
}, {
  dependsOn: [localNetworkSourceAsset],
});

// TODO: figure out way to differentiate between copying a directory vs the
// contents of a directory (like all of the cp/rsync trailing slash behavior)
const localSourceDirectoryArchive = new mid.resource.File("local-source-directory-archive", {
  path: "/tmp/local-source-directory-archive/",
  source: new pulumi.asset.FileArchive(__dirname + "/testdata/testdir/"),
  // FIXME: default mode should come from umask
  mode: "a+rx",
  recurse: true,
}, {
  provider,
  deleteBeforeReplace: true,
});

new remote.Command("local-source-directory-archive", {
  connection,
  create: testdirAssertion("local-source-directory-archive"),
}, {
  dependsOn: [localSourceDirectoryArchive],
});

const localSourceTarGzArchive = new mid.resource.File("local-source-targz-archive", {
  path: "/tmp/local-source-targz-archive",
  source: new pulumi.asset.FileArchive(__dirname + "/testdata/testdir.tar.gz"),
  // FIXME: default mode should come from umask
  mode: "a+rx",
  recurse: true,
}, {
  provider,
});

new remote.Command("local-source-targz-archive", {
  connection,
  create: testdirAssertion("local-source-targz-archive/testdir"),
}, {
  dependsOn: [localSourceTarGzArchive],
});

// FIXME: flaky
// const localSourceZipArchive = new mid.resource.File("local-source-zip-archive", {
//   path: "/tmp/local-source-zip-archive",
//   source: new pulumi.asset.FileArchive(__dirname + "/testdata/testdir.zip"),
//   // FIXME: default mode should come from umask
//   mode: "a+rx",
//   recurse: true,
// }, {
//   provider,
// });
//
// new remote.Command("local-source-zip-archive", {
//   connection,
//   create: testdirAssertion("local-source-zip-archive/testdir"),
// }, {
//   dependsOn: [localSourceZipArchive],
// });

const localNetworkSourceTarGzArchive = new mid.resource.File("local-network-source-targz-archive", {
  path: "/tmp/local-network-source-targz-archive",
  source: new pulumi.asset.RemoteArchive(
    "https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testdir.tar.gz",
  ),
  // FIXME: default mode should come from umask
  mode: "a+rx",
  recurse: true,
}, {
  provider,
});

new remote.Command("local-network-source-targz-archive", {
  connection,
  create: testdirAssertion("local-network-source-targz-archive/testdir"),
}, {
  dependsOn: [localNetworkSourceTarGzArchive],
});

const localNetworkSourceZipArchive = new mid.resource.File("local-network-source-zip-archive", {
  path: "/tmp/local-network-source-zip-archive",
  source: new pulumi.asset.RemoteArchive(
    "https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testdir.zip",
  ),
  // FIXME: default mode should come from umask
  mode: "a+rx",
  recurse: true,
}, {
  provider,
});

new remote.Command("local-network-source-zip-archive", {
  connection,
  create: testdirAssertion("local-network-source-zip-archive/testdir"),
}, {
  dependsOn: [localNetworkSourceZipArchive],
});

// FIXME: AssetArchive doesn't work?
// const localSourceArchiveOfAsset = new mid.resource.File("local-source-archive-of-assets", {
//   path: "/tmp/local-source-archive-of-assets",
//   source: new pulumi.asset.AssetArchive({
//     "passwords.txt": new pulumi.asset.FileAsset("./testdata/testdir/passwords.txt"),
//     "sunday.txt": new pulumi.asset.FileAsset("./testdata/testdir/sunday.txt"),
//   }),
//   mode: "a+rx",
//   recurse: true,
// }, {
//   provider,
// });
//
// new remote.Command("local-source-archive-of-assets", {
//   connection,
//   create: testdirAssertion("local-source-archive-of-assets"),
// }, {
//   dependsOn: [localSourceArchiveOfAsset],
// });

const remoteSourceFile = new mid.resource.File("remote-source-file", {
  path: "/tmp/remote-source-file",
  remoteSource: "/tmp/inline-content",
}, {
  provider,
  dependsOn: [
    inlineContent,
  ],
});

new remote.Command("remote-source-file", {
  connection,
  create: testfileAssertion("remote-source-file"),
}, {
  dependsOn: [remoteSourceFile],
});

const remoteSourceDirectory = new mid.resource.File("remote-source-directory", {
  path: "/tmp/remote-source-directory/",
  remoteSource: "/tmp/local-source-directory-archive/",
}, {
  provider,
  dependsOn: [
    localSourceDirectoryArchive,
  ],
});

new remote.Command("remote-source-directory", {
  connection,
  create: testdirAssertion("remote-source-directory"),
}, {
  dependsOn: [remoteSourceDirectory],
});

const remoteNetworkSourceFile = new mid.resource.File("remote-network-source-file", {
  path: "/tmp/remote-network-source-file",
  remoteSource: "https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testfile",
}, {
  provider,
});

new remote.Command("remote-network-source-file", {
  connection,
  create: testfileAssertion("remote-network-source-file"),
}, {
  dependsOn: [remoteNetworkSourceFile],
});

const remoteNetworkSourceTarGzArchive = new mid.resource.File("remote-network-source-targz-archive", {
  path: "/tmp/remote-network-source-targz-archive",
  remoteSource: "https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testdir.tar.gz",
  ensure: "directory",
}, {
  provider,
  dependsOn: [
    localSourceDirectoryArchive,
  ],
});

new remote.Command("remote-network-source-targz-archive", {
  connection,
  create: testdirAssertion("remote-network-source-targz-archive/testdir"),
}, {
  dependsOn: [remoteNetworkSourceTarGzArchive],
});

const remoteNetworkSourceZipArchive = new mid.resource.File("remote-network-source-zip-archive", {
  path: "/tmp/remote-network-source-zip-archive",
  remoteSource: "https://sapslaj-stuff.s3.us-east-1.amazonaws.com/mid-testdata/testdir.zip",
  ensure: "directory",
}, {
  provider,
  dependsOn: [
    unzip,
    localSourceDirectoryArchive,
  ],
});

new remote.Command("remote-network-source-zip-archive", {
  connection,
  create: testdirAssertion("remote-network-source-zip-archive/testdir"),
}, {
  dependsOn: [remoteNetworkSourceZipArchive],
});
