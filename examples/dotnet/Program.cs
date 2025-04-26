using System.Collections.Generic;
using System.Linq;
using Pulumi;
using mid = Pulumi.mid;

return await Deployment.RunAsync(() => 
{
    var myRandomResource = new mid.Random("myRandomResource", new()
    {
        Length = 24,
    });

    return new Dictionary<string, object?>
    {
        ["output"] = 
        {
            { "value", myRandomResource.Result },
        },
    };
});

