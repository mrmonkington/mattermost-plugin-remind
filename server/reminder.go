package main

import (
	"fmt"
	"time"
	"strings"
	"encoding/json"

	"github.com/mattermost/mattermost-server/model"
)

type Reminder struct {
	TeamId string

	Id string

	Username string

	Target string

	Message string

	When string

	Occurrences []ReminderOccurrence

	Completed time.Time
}

type ReminderRequest struct {
	TeamId string

	Username string

	Payload string

	Reminder Reminder
}

func (p *Plugin) UpsertReminder(request ReminderRequest) {

	user, u_err := p.API.GetUserByUsername(request.Username)

	if u_err != nil {
		p.API.LogError("failed to query user %s", request.Username)
		return
	}

	bytes, b_err := p.API.KVGet(user.Username)
	if b_err != nil {
		p.API.LogError("failed KVGet %s", b_err)
		return
	}

	var reminders []Reminder
	err := json.Unmarshal(bytes, &reminders)

	if err != nil {
		p.API.LogError("new reminder " + user.Username)
	} else {
		p.API.LogError("existing " + fmt.Sprintf("%v", reminders))
	}

	reminders = append(reminders, request.Reminder)
	ro, __ := json.Marshal(reminders)

	if __ != nil {
		p.API.LogError("failed to marshal reminders %s", user.Username)
		return
	}

	p.API.KVSet(user.Username, ro)
}

func (p *Plugin) TriggerReminders() {

	bytes, err := p.API.KVGet(string(fmt.Sprintf("%v", time.Now().Round(time.Second))))

	if err != nil {
		p.API.LogError("failed KVGet %s", err)
	} else if string(bytes[:]) == "" {
	} else {

		var reminderOccurrences []ReminderOccurrence

		roErr := json.Unmarshal(bytes, &reminderOccurrences)
		if roErr != nil {
			p.API.LogError("Failed to unmarshal reminder occurrences " + fmt.Sprintf("%v", roErr))
			return
		}

		p.API.LogError("existing " + fmt.Sprintf("%v", reminderOccurrences))

		// TODO loop through array of occurrences, and trigger DM between remind user & user
		for _, ReminderOccurrence := range reminderOccurrences {

			user, err := p.API.GetUserByUsername(ReminderOccurrence.Username)

			if err != nil {
				p.API.LogError("failed to query user %s", user.Id)
				continue
			}

			bytes, b_err := p.API.KVGet(user.Username)
			if b_err != nil {
				p.API.LogError("failed KVGet %s", b_err)
				return
			}

			var reminders []Reminder
			uerr := json.Unmarshal(bytes, &reminders)

			if uerr != nil {
				continue
			}

			//var reminder Reminder
			reminder := p.findReminder(reminders, ReminderOccurrence)

			p.API.LogError(fmt.Sprintf("%v", reminder))

			if strings.HasPrefix(reminder.Target, "@") || strings.HasPrefix(reminder.Target, "me") {

				p.API.LogError(fmt.Sprintf("%v", p.remindUserId) + " " + fmt.Sprintf("%v", user.Id))
				channel, cerr := p.API.GetDirectChannel(p.remindUserId, user.Id)

				if cerr != nil {
					p.API.LogError("fail to get direct channel ", fmt.Sprintf("%v", cerr))
				} else {
					p.API.LogError("got direct channel " + fmt.Sprintf("%v", channel))

					var finalTarget string
					finalTarget = reminder.Target
					if finalTarget == "me" {
						finalTarget = "You"
					} else {
						finalTarget = "@" + user.Username
					}

					if _, err = p.API.CreatePost(&model.Post{
						UserId:    p.remindUserId,
						ChannelId: channel.Id,
						Message:   fmt.Sprintf(finalTarget + " asked me to remind you \"" + reminder.Message + "\"."),
					}); err != nil {
						p.API.LogError(
							"failed to post DM message",
							"user_id", user.Id,
							"error", err.Error(),
						)
					}
				}

			} else { //~ channel

				channel, cerr := p.API.GetChannelByName(reminder.TeamId, strings.Replace(reminder.Target, "~", "", -1), false)

				if cerr != nil {
					p.API.LogError("fail to get channel " + fmt.Sprintf("%v", cerr))
				} else {
					p.API.LogDebug("got channel " + fmt.Sprintf("%v", channel))

					if _, err = p.API.CreatePost(&model.Post{
						UserId:    p.remindUserId,
						ChannelId: channel.Id,
						Message:   fmt.Sprintf("@" + user.Username + " asked me to remind you \"" + reminder.Message + "\"."),
					}); err != nil {
						p.API.LogError(
							"failed to post DM message",
							"user_id", user.Id,
							"error", err.Error(),
						)
					}
				}
			}

		}

	}

}

func (p *Plugin) findReminder(reminders []Reminder, reminderOccurrence ReminderOccurrence) (Reminder) {
	for _, reminder := range reminders {
		if reminder.Id == reminderOccurrence.ReminderId {
			p.API.LogError("FOUND!")
			return reminder
		}
	}
	return Reminder{}
}