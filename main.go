package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/pterm/pterm"
	"github.com/spf13/pflag"
)

var (
	s *discordgo.Session

	// registered commands
	commands = []*discordgo.ApplicationCommand{{
		Name:        "devbadge",
		Type:        discordgo.ChatApplicationCommand,
		Description: "Activate your discord developer badge",
	}}

	// handlers for registered commands
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"devbadge": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			msg := "Command received. Please check https://discord.com/developers/active-developer to redeem your badge. It can take up to 24h to show up."
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
			spinnerSlashCmd.Success(msg)
			stopBot()
		}}

	// console spinners
	spinnerSlashCmd   *pterm.SpinnerPrinter
	spinnerJoinServer *pterm.SpinnerPrinter
)

func init() {
	// parse cli args
	var token string
	pflag.StringVarP(&token, "token", "t", "", "The Discord bot token you got from the developer portal")
	pflag.Parse()

	if token == "" {
		pterm.Error.Println("Required token missing. Usage:")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	// create discord bot
	var err error
	s, err = discordgo.New("Bot " + token)
	if err != nil {
		pterm.Fatal.Println(err.Error())
	}
}

func stopBot() {
	// stops the bot
	pterm.Debug.Println("Bot stopped")
	os.Exit(0)
}

func main() {
	spinnerJoinServer, _ = pterm.DefaultSpinner.Start("Waiting to join server...")

	// debug login info
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		pterm.Debug.Printfln("Logged in as: %s#%s", s.State.User.Username, s.State.User.Discriminator)
	})

	// print join server
	s.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		spinnerJoinServer.Success(fmt.Sprintf("Joined server: \"%s\"", g.Name))
		spinnerSlashCmd, _ = pterm.DefaultSpinner.Start("Waiting for \"/devbadge\" command...")
	})

	// print leave server
	s.AddHandler(func(s *discordgo.Session, g *discordgo.GuildDelete) {
		pterm.Warning.Printfln("Left server: \"%s\"", g.BeforeDelete.Name)
	})

	// add handlers
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	// run bot
	if err := s.Open(); err != nil {
		pterm.Fatal.Println(err.Error())
	}

	// register commands
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			pterm.Fatal.Printfln("Cannot create '%v' command: %v", v.Name, err.Error())
		}
		registeredCommands[i] = cmd
	}

	var msg string
	if len(s.State.Guilds) > 0 {
		spinnerJoinServer.Success(fmt.Sprintf("Joined servers: %d", len(s.State.Guilds)))
		msg = fmt.Sprintf("Already joined a server. To join another server, open: https://discord.com/oauth2/authorize?client_id=%s&scope=applications.commands%%20bot&permissions=3072", s.State.User.ID)
	} else {
		msg = fmt.Sprintf("To join a new server, open: https://discord.com/oauth2/authorize?client_id=%s&scope=applications.commands%%20bot&permissions=3072", s.State.User.ID)
	}
	pterm.Info.Println(msg)
	pterm.Debug.Println("Bot running. Press ctrl+c to exit.")
	fmt.Println()

	defer s.Close()

	// wait for kill
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	stopBot()
}
