package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/spf13/cobra"
)

type GroupMembership struct {
	GroupName string
	Members   []string
}

// generateGroupMembershipsOptimized generates group memberships in O(n) time
// by iterating through users once and determining their group memberships
func generateGroupMembershipsOptimized(userDNs []string, numGroups int) []GroupMembership {
	rand.Seed(time.Now().UnixNano())
	groups := make([]GroupMembership, numGroups+1) // +1 for developers group

	// Initialize groups
	groups[0] = GroupMembership{
		GroupName: "developers",
		Members:   make([]string, 0, len(userDNs)),
	}
	for i := 1; i <= numGroups; i++ {
		groups[i] = GroupMembership{
			GroupName: fmt.Sprintf("group-%d", i),
			Members:   make([]string, 0, len(userDNs)/3), // Estimate capacity based on 40% membership
		}
	}

	// Single pass through users to determine all group memberships
	for _, userDN := range userDNs {
		// All users are in developers group
		groups[0].Members = append(groups[0].Members, userDN)

		// 10% chance to be in each additional group
		for i := 1; i <= numGroups; i++ {
			if rand.Float64() < 0.1 {
				groups[i].Members = append(groups[i].Members, userDN)
			}
		}
	}

	// Ensure each additional group has at least one member
	for i := 1; i <= numGroups; i++ {
		if len(groups[i].Members) == 0 {
			groups[i].Members = append(groups[i].Members, userDNs[0])
		}
	}

	return groups
}

// writeGroupToLDIF writes a single group's LDIF entry to the writer
func writeGroupToLDIF(writer *bufio.Writer, group GroupMembership, baseDN string) error {
	// Generate group entry
	groupDescription := "Development team"
	if group.GroupName != "developers" {
		groupDescription = fmt.Sprintf("Group %s members", group.GroupName)
	}

	// Write group header
	_, err := fmt.Fprintf(writer, "# Group: %s\ndn: cn=%s,ou=groups,%s\nobjectClass: groupOfNames\ncn: %s\ndescription: %s\n",
		group.GroupName, group.GroupName, baseDN, group.GroupName, groupDescription)
	if err != nil {
		return err
	}

	// Write members
	for _, member := range group.Members {
		_, err = fmt.Fprintf(writer, "member: %s\n", member)
		if err != nil {
			return err
		}
	}

	// Add blank line after group
	_, err = writer.WriteString("\n")
	return err
}

// generateLDIFToFile generates LDIF content directly to a file using streaming writes
func generateLDIFToFile(startIndex, endIndex int, baseDN, userPassword string, numGroups int, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write LDIF header
	_, err = fmt.Fprintf(writer, `# Mattermost Users LDIF Export
# Generated on %s

# Create users organizational unit
dn: ou=users,%s
objectClass: organizationalUnit
ou: users

# Create groups organizational unit
dn: ou=groups,%s
objectClass: organizationalUnit
ou: groups
description: Container for group accounts

`, time.Now().Format(time.RFC822), baseDN, baseDN)
	if err != nil {
		return fmt.Errorf("failed to write LDIF header: %w", err)
	}

	// Generate users and collect user DNs for group membership
	var userDNs []string
	for i := startIndex; i <= endIndex; i++ {
		username := fmt.Sprintf("testuser-%d", i)
		userDN := fmt.Sprintf("uid=%s,ou=users,%s", username, baseDN)
		userDNs = append(userDNs, userDN)

		// Write user entry directly to file
		_, err = fmt.Fprintf(writer, `# User: %s
dn: %s
objectClass: inetOrgPerson
objectClass: person
objectClass: top
uid: %s
cn: %s
sn: User
mail: %s@mattermost.com
userPassword: %s

`, username, userDN, username, username, username, userPassword)
		if err != nil {
			return fmt.Errorf("failed to write user %s: %w", username, err)
		}
	}

	// Generate group memberships in O(n) time
	groupMemberships := generateGroupMembershipsOptimized(userDNs, numGroups)

	// Write each group directly to file
	for _, group := range groupMemberships {
		if err := writeGroupToLDIF(writer, group, baseDN); err != nil {
			return fmt.Errorf("failed to write group %s: %w", group.GroupName, err)
		}
	}

	return nil
}

func RunGenerateUsersCommandF(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected exactly 2 arguments: start-index and end-index")
	}

	startIndex, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid start index: %w", err)
	}

	endIndex, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid end index: %w", err)
	}

	if startIndex > endIndex {
		return fmt.Errorf("start index (%d) must be less than or equal to end index (%d)", startIndex, endIndex)
	}

	if startIndex < 1 {
		return fmt.Errorf("start index must be at least 1")
	}

	// Get configuration
	deployerConfigPath, err := cmd.Flags().GetString("deployer-config")
	if err != nil {
		return fmt.Errorf("failed to read config flag: %w", err)
	}

	if deployerConfigPath == "" {
		return fmt.Errorf("config flag is required")
	}

	// Read deployment config
	deploymentConfig, err := deployment.ReadConfig(deployerConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read deployment configuration from %s: %w", deployerConfigPath, err)
	}

	// Use configuration values
	baseDN := deploymentConfig.OpenLDAPSettings.BaseDN
	bindDN := deploymentConfig.OpenLDAPSettings.BindUsername
	bindPassword := deploymentConfig.OpenLDAPSettings.BindPassword

	if baseDN == "" {
		return fmt.Errorf("OpenLDAP BaseDN not configured in deployment config")
	}
	if bindDN == "" {
		return fmt.Errorf("OpenLDAP BindUsername not configured in deployment config")
	}
	if bindPassword == "" {
		return fmt.Errorf("OpenLDAP BindPassword not configured in deployment config")
	}

	userPassword, err := cmd.Flags().GetString("user-password")
	if err != nil {
		return fmt.Errorf("failed to read user-password flag: %w", err)
	}
	if userPassword == "" {
		userPassword = "testPass123$"
	}

	outputFile, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return fmt.Errorf("failed to read output-file flag: %w", err)
	}
	if outputFile == "" {
		outputFile = fmt.Sprintf("users_%d_%d.ldif", startIndex, endIndex)
	}

	numGroups, err := cmd.Flags().GetInt("num-groups")
	if err != nil {
		return fmt.Errorf("failed to read num-groups flag: %w", err)
	}
	if numGroups < 0 {
		return fmt.Errorf("num-groups must be non-negative, got: %d", numGroups)
	}

	// Generate LDIF content
	mlog.Info("Generating LDIF file",
		mlog.Int("start_index", startIndex),
		mlog.Int("end_index", endIndex),
		mlog.String("output_file", outputFile),
		mlog.String("base_dn", baseDN),
		mlog.Int("num_groups", numGroups+1)) // +1 for developers group

	// Generate LDIF directly to file using streaming writes
	err = generateLDIFToFile(startIndex, endIndex, baseDN, userPassword, numGroups, outputFile)
	if err != nil {
		return fmt.Errorf("failed to generate LDIF file: %w", err)
	}

	mlog.Info("LDIF file generated successfully", mlog.String("file", outputFile))

	// Import to LDAP if requested
	importToLDAP, err := cmd.Flags().GetBool("import")
	if err != nil {
		return fmt.Errorf("failed to read import flag: %w", err)
	}

	if importToLDAP {
		var ldapHost string
		t, err := terraform.New("", *deploymentConfig)
		if err != nil {
			return fmt.Errorf("failed to create terraform client: %w", err)
		}
		terraformOutput, err := t.Output()
		if err != nil {
			return fmt.Errorf("failed to get terraform output: %w", err)
		}
		if terraformOutput.HasOpenLDAP() {
			ldapHost = terraformOutput.OpenLDAPServer.GetConnectionIP()
		}

		if ldapHost == "" {
			return fmt.Errorf("LDAP host not specified and could not be determined from terraform output")
		}

		// Import to LDAP using ldapadd
		mlog.Info("Importing LDIF to LDAP server",
			mlog.String("host", ldapHost),
			mlog.String("bind_dn", bindDN))

		ldapaddCmd := exec.Command("ldapadd",
			"-x",
			"-H", fmt.Sprintf("ldap://%s:389", ldapHost),
			"-D", bindDN,
			"-w", bindPassword,
			"-f", outputFile,
			"-c") // Continue on errors

		err = terraform.RunCommand(ldapaddCmd, nil)
		if err != nil {
			mlog.Error("Failed to import LDIF", mlog.Err(err))
			return fmt.Errorf("failed to import LDIF to LDAP: %w", err)
		}

		mlog.Info("Successfully imported LDIF to LDAP server")
	}

	return nil
}

func MakeGenerateUsersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users <start-index> <end-index>",
		Short: "Generate LDIF file with users from start-index to end-index",
		Long: `Generate an LDIF file containing users from testuser-<start> to testuser-<end>.
All users will have the password 'testPass123$' and will be part of the 'developers' group.
Additional groups (group-1, group-2, etc.) can be created with random user membership.
LDAP connection settings (BaseDN, BindUsername, BindPassword) are read from the deployment config.

Examples:
  ltldap generate users 1 100 --deployer-config deployer.json                    # Generate users with developers group only
  ltldap generate users 1 100 --deployer-config deployer.json --num-groups 3    # Generate users with developers + 3 additional groups
  ltldap generate users 1 10 --deployer-config deployer.json --import            # Generate and import to LDAP
  ltldap generate users 1 5 --deployer-config deployer.json --num-groups 2 --import  # Generate with groups and import`,
		Args: cobra.ExactArgs(2),
		RunE: RunGenerateUsersCommandF,
	}

	cmd.Flags().String("deployer-config", "", "Path to the deployer configuration file (required)")
	cmd.Flags().String("user-password", "testPass123$", "Password for all generated users")
	cmd.Flags().String("output-file", "", "Output LDIF file name (defaults to users_<start>_<end>.ldif)")
	cmd.Flags().Bool("import", false, "Import the generated LDIF to LDAP server")
	cmd.Flags().Int("num-groups", 0, "Number of additional groups to create (beyond developers group)")

	return cmd
}
