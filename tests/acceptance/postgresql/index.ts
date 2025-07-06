import * as crypto from "crypto";

import * as aws from "@pulumi/aws";
import { remote } from "@pulumi/command";
import * as postgresql from "@pulumi/postgresql";
import * as pulumi from "@pulumi/pulumi";
import * as random from "@pulumi/random";
import * as tls from "@pulumi/tls";
import * as mid from "@sapslaj/pulumi-mid";

// world's second most insecure security group (just for our own sanity)
const sg = new aws.ec2.SecurityGroup("postgresql", {
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
    // protip: don't actually allow inbound SSH and PostgreSQL ports from
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

// generate new SSH private key
const privateKey = new tls.PrivateKey("postgresql", {
  algorithm: "ED25519",
});

// register that key with AWS
const keypair = new aws.ec2.KeyPair("postgresql", {
  publicKey: privateKey.publicKeyOpenssh,
});

// IAM role for the instance
const role = new aws.iam.Role("postgresql", {
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

const instanceProfile = new aws.iam.InstanceProfile("postgresql", {
  name: role.name,
  role: role.name,
});

// now for the instance (using Ubuntu 24.04)
const instance = new aws.ec2.Instance("postgresql", {
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
        values: ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"],
      },
    ],
  }).id,
  iamInstanceProfile: instanceProfile.name,
  instanceType: "t3a.micro",
  vpcSecurityGroupIds: [sg.id],
  keyName: keypair.keyName,
}, {});

// setup mid provider
const provider = new mid.Provider("postgresql", {
  connection: {
    host: instance.publicIp,
    user: "ubuntu",
    privateKey: privateKey.privateKeyOpenssh,
  },
});

// configure apt to always `--no-install-recommends`
const aptNoInstallRecommends = new mid.resource.File("/etc/apt/apt.conf.d/999norecommend", {
  path: "/etc/apt/apt.conf.d/999norecommend",
  content: [
    `APT::Install-Recommends "0";\n`,
    `APT::Install-Suggests "0";\n`,
  ].join(""),
}, {
  provider,
});

// install `python3-apt` and never delete it from the system since this is
// needed for `Apt` resources to work correctly without `forceAptGet`.
const python3Apt = new mid.resource.Apt("python3-apt", {
  forceAptGet: true,
  updateCache: true,
  name: "python3-apt",
}, {
  provider,
  retainOnDelete: true,
  dependsOn: [
    aptNoInstallRecommends,
  ],
});

// install pgdg dependencies making sure `python3-apt` is installed and apt
// uses `--no-install-recommends` as explicit Pulumi dependencies
const pgdgDeps = new mid.resource.Apt("pgdg-deps", {
  names: [
    "curl",
    "ca-certificates",
  ],
}, {
  provider,
  dependsOn: [
    python3Apt,
  ],
});

// mkdir -p /usr/share/postgresql-common/pgdg/
const usrSharePostgresqlCommonPgdg = new mid.resource.File("/usr/share/postgresql-common/pgdg/", {
  path: "/usr/share/postgresql-common/pgdg/",
  ensure: "directory",
}, {
  provider,
});

// curl https://www.postgresql.org/media/keys/ACCC4CF8.asc -O /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc
const pgdgSig = new mid.resource.File("/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc", {
  path: "/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc",
  source: new pulumi.asset.RemoteAsset("https://www.postgresql.org/media/keys/ACCC4CF8.asc"),
}, {
  provider,
  dependsOn: [
    usrSharePostgresqlCommonPgdg,
  ],
});

// set up apt source
const pgdgAptSource = new mid.resource.File("/etc/apt/sources.list.d/pgdg.list", {
  path: "/etc/apt/sources.list.d/pgdg.list",
  content:
    "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt noble-pgdg main\n",
}, {
  provider,
  dependsOn: [
    pgdgSig,
  ],
});

// install PostgreSQL!
const postgresqlPackage = new mid.resource.Apt("postgresql", {
  // `updateCache` is needed after adding the apt source
  updateCache: true,
  name: "postgresql",
}, {
  provider,
  dependsOn: [
    pgdgDeps,
    pgdgSig,
    pgdgAptSource,
    python3Apt,
  ],
});

// apt will install the latest version by default, so this command figures out
// what version _did_ get installed.
const postgresqlVersion = mid.agent.execOutput({
  command: [
    "/bin/sh",
    "-c",
    `basename "$(find /etc/postgresql -mindepth 1 -maxdepth 1 -type d | head -n 1)"`,
  ],
}, {
  provider,
  dependsOn: [
    postgresqlPackage,
  ],
}).stdout.apply((stdout) => stdout.trim());

// reduce the default max_connections (not really necessary but just here for
// an example)
const postgresqlMaxConnections = new mid.resource.FileLine("postgresql-max-connections", {
  path: pulumi.interpolate`/etc/postgresql/${postgresqlVersion}/main/postgresql.conf`,
  line: "max_connections = 50",
  regexp: "^max_connections ",
}, {
  provider,
  dependsOn: [
    postgresqlPackage,
  ],
});

// open up listen_addresses
const postgresqlListenAddresses = new mid.resource.FileLine("postgresql-listen-addresses", {
  path: pulumi.interpolate`/etc/postgresql/${postgresqlVersion}/main/postgresql.conf`,
  line: "listen_addresses = '*'",
  regexp: "^listen_addresses ",
}, {
  provider,
  dependsOn: [
    // FIXME: cannot edit the same file concurrently or else the changes get
    // eaten. need to implement file locking apparently.
    postgresqlMaxConnections,
    postgresqlPackage,
  ],
});

// allow logins from anywhere
const postgresqlHBA = new mid.resource.FileLine("postgresql-hba", {
  path: pulumi.interpolate`/etc/postgresql/${postgresqlVersion}/main/pg_hba.conf`,
  line: [
    "host", // TYPE
    "all", // DATABASE
    "",
    "all", // USER
    "",
    "0.0.0.0/0", // ADDRESS
    "",
    "md5", // METHOD
  ].join("\t"),
  regexp: "0\\.0\\.0\\.0\\/0",
}, {
  provider,
  dependsOn: [
    postgresqlPackage,
  ],
});

// make sure postgresql.service is started and enabled, and restart on any
// changes to any of the triggers.
const postgresqlService = new mid.resource.SystemdService("postgresql.service", {
  name: pulumi.interpolate`postgresql@${postgresqlVersion}-main.service`,
  ensure: "started",
  enabled: true,
  triggers: {
    refresh: [
      postgresqlMaxConnections.triggers.lastChanged,
      postgresqlListenAddresses.triggers.lastChanged,
      postgresqlHBA.triggers.lastChanged,
    ],
  },
}, {
  provider,
  dependsOn: [
    postgresqlPackage,
  ],
});

// generate a new "superadmin" password
const postgresqlSuperadminPassword = new random.RandomPassword("postgresql-superadmin", {
  length: 32,
  special: false,
});

// use the Power of JavaScript to calculate the MD5
const postgresqlSuperadminPasswordMD5 = postgresqlSuperadminPassword.result.apply((password) => {
  return "md5" + crypto.createHash("md5").update(password + "superadmin").digest("hex");
});

// do some raw psql to create the "superadmin" user. have to use psql directly
// here since we only have the `postgres` user.
const postgresqlSuperadmin = new mid.resource.Exec("postgresql-superadmin", {
  expandArgumentVars: true,
  environment: {
    PGPASSWORD_MD5: postgresqlSuperadminPasswordMD5,
  },
  create: {
    command: [
      "sudo",
      "-u",
      "postgres",
      "psql",
      "-c",
      `
        CREATE USER superadmin WITH
          SUPERUSER
          CREATEDB
          CREATEROLE
          LOGIN
          REPLICATION
          ENCRYPTED PASSWORD '$PGPASSWORD_MD5'
        ;
      `,
    ],
  },
  update: {
    command: [
      "sudo",
      "-u",
      "postgres",
      "psql",
      "-c",
      `
        ALTER USER superadmin WITH
          SUPERUSER
          CREATEDB
          CREATEROLE
          LOGIN
          REPLICATION
          ENCRYPTED PASSWORD '$PGPASSWORD_MD5'
        ;
      `,
    ],
  },
  delete: {
    command: [
      "sudo",
      "-u",
      "postgres",
      "psql",
      "-c",
      pulumi.interpolate`
        DROP USER IF EXISTS superadmin;
      `,
    ],
  },
}, {
  provider,
  dependsOn: [
    postgresqlService,
  ],
});

// just grabbing some logs for debugging test failures.
export const debuglogs = new remote.Command("debuglogs", {
  connection: {
    host: instance.publicIp,
    user: "ubuntu",
    privateKey: privateKey.privateKeyOpenssh,
  },
  create:
    `bash -xc "sudo journalctl -xe --no-pager | tail -n 20 ; sudo systemctl status postgresql.service ; sudo ss -tlpn" 2>&1`,
}, {
  dependsOn: [
    postgresqlService,
  ],
}).stdout.apply((stdout) => {
  console.log(stdout);
  return stdout;
});

// set up the postgresql Pulumi provider
const postgresqlProvider = new postgresql.Provider("postgresql", {
  host: instance.publicIp,
  username: "superadmin",
  password: postgresqlSuperadminPassword.result,
}, {
  dependsOn: [
    postgresqlSuperadmin,
  ],
});

// create new fake "app" user
const appRole = new postgresql.Role("app", {
  name: "app",
  password: "hunter2",
}, {
  provider: postgresqlProvider,
});

// create new fake "app" database
const appDB = new postgresql.Database("app", {
  name: "app",
  owner: appRole.name,
}, {
  provider: postgresqlProvider,
});

// create new fake "app" schema
const appSchema = new postgresql.Schema("app", {
  name: "app",
  database: appDB.name,
  owner: appRole.name,
}, {
  provider: postgresqlProvider,
});
