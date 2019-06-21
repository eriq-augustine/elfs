package driver;

// Get a driver from the command line.

import (
    "encoding/hex"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/pkg/errors"
    "github.com/spf13/pflag"

    "github.com/eriq-augustine/elfs/connector"
)

const (
    DEFAULT_AWS_CRED_PATH = "config/elfs-wasabi-credentials"
    DEFAULT_AWS_ENDPOINT = ""
    DEFAULT_AWS_PROFILE = "elfsapi"
    DEFAULT_AWS_REGION = "us-east-1"
)

// This is meant to be called from a command line.
// This will just exit on bad args.
// The caller is responsible for closing the driver when done.
func GetDriverFromArgs() (*Driver, *Args) {
    args, err := parseArgs();
    if (err != nil) {
        pflag.Usage();
        fmt.Printf("Error parsing args: %+v\n", err);
        os.Exit(1);
    }

    var fsDriver *Driver = nil;
    if (args.ConnectorType == connector.CONNECTOR_TYPE_LOCAL) {
        fsDriver, err = NewLocalDriver(args.Key, args.IV, args.Path, args.Force);
        if (err != nil) {
            fmt.Printf("%+v\n", errors.Wrap(err, "Failed to get local driver"));
            os.Exit(2);
        }
    } else if (args.ConnectorType == connector.CONNECTOR_TYPE_S3) {
        fsDriver, err = NewS3Driver(args.Key, args.IV, args.Path, args.AwsCredPath, args.AwsProfile, args.AwsRegion, args.AwsEndpoint, args.Force);
        if (err != nil) {
            fmt.Printf("%+v\n", errors.Wrap(err, "Failed to get S3 driver"));
            os.Exit(3);
        }
    } else {
        fmt.Printf("Unknown connector type: [%s]\n", args.ConnectorType);
        os.Exit(4);
    }

    // Gracefully handle SIGINT and SIGTERM.
    sigChan := make(chan os.Signal, 1);
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM);
    go func() {
        <-sigChan;
        fsDriver.Close();
        os.Exit(0);
    }();

    return fsDriver, args;
}

func parseArgs() (*Args, error) {
    var awsCredPath *string = pflag.StringP("aws-creds", "c", DEFAULT_AWS_CRED_PATH, "Path to AWS credentials");
    var awsEndpoint *string = pflag.StringP("aws-endpoint", "e", DEFAULT_AWS_ENDPOINT, "AWS endpoint to use. Empty string uses standard AWS S3, 'https://s3.wasabisys.com' uses Wasabi, etc..");
    var awsProfile *string = pflag.StringP("aws-profile", "l", DEFAULT_AWS_PROFILE, "AWS profile to use");
    var awsRegion *string = pflag.StringP("aws-region", "r", DEFAULT_AWS_REGION, "AWS region to use");
    var connectorType *string = pflag.StringP("type", "t", "", "Connector type ('s3' or 'local')");
    var hexKey *string = pflag.StringP("key", "k", "", "Encryption key in hex");
    var hexIV *string = pflag.StringP("iv", "i", "", "IV in hex");
    var path *string = pflag.StringP("path", "p", "", "Path to the filesystem");
    var user *string = pflag.StringP("user", "u", "root", "User to login as");
    var pass *string = pflag.StringP("password", "w", "", "Password to use for login");
    var force *bool = pflag.BoolP("force", "f", false, "Force the filesystem to mount regardless of locks");
    var mountpoint *string = pflag.StringP("mountpoint", "m", "", "The mountpoint of the filesystem (not used by all operations)");

    pflag.Parse();

    if (hexKey == nil || *hexKey == "") {
        return nil, errors.New("Error: Key required.");
    }

    if (hexIV == nil || *hexIV == "") {
        return nil, errors.New("Error: IV required.");
    }

    if (connectorType == nil || *connectorType == "") {
        // Can't take the address of a constant.
        var tempType string = connector.CONNECTOR_TYPE_LOCAL;
        connectorType = &tempType;
    }

    if (path == nil || *path == "") {
        return nil, errors.New("Error: Path required.");
    }

    key, err := hex.DecodeString(*hexKey);
    if (err != nil) {
        return nil, errors.Wrap(err, "Could not decode hex key.");
    }

    iv, err := hex.DecodeString(*hexIV);
    if (err != nil) {
        return nil, errors.Wrap(err, "Could not decode hex iv.");
    }

    var rtn Args = Args{
        AwsCredPath: *awsCredPath,
        AwsEndpoint: *awsEndpoint,
        AwsProfile: *awsProfile,
        AwsRegion: *awsRegion,
        ConnectorType: *connectorType,
        Key: key,
        IV: iv,
        Path: *path,
        User: *user,
        Pass: *pass,
        Force: *force,
        Mountpoint: *mountpoint,
    };

    return &rtn, nil;
}

type Args struct {
    AwsCredPath string
    AwsEndpoint string
    AwsProfile string
    AwsRegion string
    ConnectorType string
    Key []byte
    IV []byte
    Path string
    User string
    Pass string
    Force bool
    Mountpoint string
}
