package deployment

type Server struct {
	Count        int
	InstanceType string
}

type ReferenceArchitecture struct {
	ActiveUsers    int
	AppServers     Server
	DatabaseServer Server
}

var Architectures = map[int]ReferenceArchitecture{
	100: {
		ActiveUsers: 100,
		AppServers: Server{
			Count:        1,
			InstanceType: "c6i.large",
		},
		DatabaseServer: Server{
			Count:        1,
			InstanceType: "db.r6g.large",
		},
	},
}
