package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// StaticCampaign is a struct representing a created static campaign
type StaticCampaign struct {
	Id            int64     `json:"id"`
	UserId        int64     `json:"-"`
	Name          string    `json:"name" sql:"not null"`
	CreatedDate   time.Time `json:"created_date"`
	LaunchDate    time.Time `json:"launch_date"`
	CompletedDate time.Time `json:"completed_date"`
	PageId        int64     `json:"-"`
	Page          Page      `json:"page"`
	Status        string    `json:"status"`
	Results       []Result  `json:"results,omitempty"`
	Events        []Event   `json:"timeline,omitemtpy"`
	URL           string    `json:"url"`
}

// StaticCampaignResults is a struct representing the results from a campaign
type StaticCampaignResults struct {
	Id      int64    `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Results []Result `json:"results,omitempty"`
	Events  []Event  `json:"timeline,omitempty"`
}

// StaticCampaignSummaries is a struct representing the overview of campaigns
type StaticCampaignSummaries struct {
	Total     int64                   `json:"total"`
	Campaigns []StaticCampaignSummary `json:"campaigns"`
}

// StaticCampaignSummary is a struct representing the overview of a single camaign
type StaticCampaignSummary struct {
	Id            int64               `json:"id"`
	CreatedDate   time.Time           `json:"created_date"`
	LaunchDate    time.Time           `json:"launch_date"`
	CompletedDate time.Time           `json:"completed_date"`
	Status        string              `json:"status"`
	Name          string              `json:"name"`
	Stats         StaticCampaignStats `json:"stats"`
}

// StaticCampaignStats is a struct representing the statistics for a single campaign
type StaticCampaignStats struct {
	Total         int64 `json:"total"`
	ClickedLink   int64 `json:"clicked"`
	SubmittedData int64 `json:"submitted_data"`
	Error         int64 `json:"error"`
}

// Validate checks to make sure there are no invalid fields in a submitted campaign
func (c *StaticCampaign) Validate() error {
	switch {
	case c.Name == "":
		return ErrCampaignNameNotSpecified
	case c.Page.Name == "":
		return ErrPageNotSpecified
	case !c.LaunchDate.IsZero():
		return ErrInvalidSendByDate
	}
	return nil
}

// UpdateStatus changes the campaign status appropriately
func (c *StaticCampaign) UpdateStatus(s string) error {
	// This could be made simpler, but I think there's a bug in gorm
	return db.Table("campaigns").Where("id=?", c.Id).Update("status", s).Error
}

// AddEvent creates a new campaign event in the database
func (c *StaticCampaign) AddEvent(e *Event) error {
	e.CampaignId = c.Id
	e.Time = time.Now().UTC()
	return db.Save(e).Error
}

// getDetails retrieves the related attributes of the campaign
// from the database. If the Events and the Results are not available,
// an error is returned. Otherwise, the attribute name is set to [Deleted],
// indicating the user deleted the attribute (template, smtp, etc.)
func (c *StaticCampaign) getDetails() error {
	err := db.Model(c).Related(&c.Results).Error
	if err != nil {
		log.Warnf("%s: results not found for campaign", err)
		return err
	}
	err = db.Model(c).Related(&c.Events).Error
	if err != nil {
		log.Warnf("%s: events not found for campaign", err)
		return err
	}
	err = db.Table("pages").Where("id=?", c.PageId).Find(&c.Page).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.Page = Page{Name: "[Deleted]"}
		log.Warnf("%s: page not found for campaign", err)
	}
	return nil
}

// getStaticCampaignStats returns a StaticCampaignStats object for the campaign with the given campaign ID.
// It also backfills numbers as appropriate with a running total, so that the values are aggregated.
func getStaticCampaignStats(cid int64) (StaticCampaignStats, error) {
	s := StaticCampaignStats{}
	query := db.Table("results").Where("campaign_id = ?", cid)
	err := query.Count(&s.Total).Error
	if err != nil {
		return s, err
	}
	query.Where("status=?", EventDataSubmit).Count(&s.SubmittedData)
	if err != nil {
		return s, err
	}
	query.Where("status=?", EventClicked).Count(&s.ClickedLink)
	if err != nil {
		return s, err
	}
	// Every submitted data event implies they clicked the link
	s.ClickedLink += s.SubmittedData

	return s, err
}

// GetStaticCampaigns returns the campaigns owned by the given user.
func GetStaticCampaigns(uid int64) ([]StaticCampaign, error) {
	cs := []StaticCampaign{}
	err := db.Model(&User{Id: uid}).Related(&cs).Error
	if err != nil {
		log.Error(err)
	}
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// GetStaticCampaignSummaries gets the summary objects for all the campaigns
// owned by the current user
func GetStaticCampaignSummaries(uid int64) (StaticCampaignSummaries, error) {
	overview := StaticCampaignSummaries{}
	cs := []StaticCampaignSummary{}
	// Get the basic campaign information
	query := db.Table("campaigns").Where("user_id = ?", uid)
	query = query.Select("id, name, created_date, completed_date, status")
	err := query.Scan(&cs).Error
	if err != nil {
		log.Error(err)
		return overview, err
	}
	for i := range cs {
		s, err := getStaticCampaignStats(cs[i].Id)
		if err != nil {
			log.Error(err)
			return overview, err
		}
		cs[i].Stats = s
	}
	overview.Total = int64(len(cs))
	overview.Campaigns = cs
	return overview, nil
}

// GetStaticCampaignSummary gets the summary object for a campaign specified by the campaign ID
func GetStaticCampaignSummary(id int64, uid int64) (StaticCampaignSummary, error) {
	cs := StaticCampaignSummary{}
	query := db.Table("campaigns").Where("user_id = ? AND id = ?", uid, id)
	query = query.Select("id, name, created_date, completed_date, status")
	err := query.Scan(&cs).Error
	if err != nil {
		log.Error(err)
		return cs, err
	}
	s, err := getStaticCampaignStats(cs.Id)
	if err != nil {
		log.Error(err)
		return cs, err
	}
	cs.Stats = s
	return cs, nil
}

// GetStaticCampaign returns the campaign, if it exists, specified by the given id and user_id.
func GetStaticCampaign(id int64, uid int64) (StaticCampaign, error) {
	c := StaticCampaign{}
	err := db.Where("id = ?", id).Where("user_id = ?", uid).Find(&c).Error
	if err != nil {
		log.Errorf("%s: campaign not found", err)
		return c, err
	}
	err = c.getDetails()
	return c, err
}

// GetStaticCampaignResults returns just the campaign results for the given campaign
func GetStaticCampaignResults(id int64, uid int64) (StaticCampaignResults, error) {
	cr := StaticCampaignResults{}
	err := db.Table("campaigns").Where("id=? and user_id=?", id, uid).Find(&cr).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"campaign_id": id,
			"error":       err,
		}).Error(err)
		return cr, err
	}
	err = db.Table("results").Where("campaign_id=? and user_id=?", cr.Id, uid).Find(&cr.Results).Error
	if err != nil {
		log.Errorf("%s: results not found for campaign", err)
		return cr, err
	}
	err = db.Table("events").Where("campaign_id=?", cr.Id).Find(&cr.Events).Error
	if err != nil {
		log.Errorf("%s: events not found for campaign", err)
		return cr, err
	}
	return cr, err
}

// GetStaticQueuedCampaigns returns the campaigns that are queued up for this given minute
func GetStaticQueuedCampaigns(t time.Time) ([]StaticCampaign, error) {
	cs := []StaticCampaign{}
	err := db.Where("launch_date <= ?", t).
		Where("status = ?", CampaignQueued).Find(&cs).Error
	if err != nil {
		log.Error(err)
	}
	log.Infof("Found %d Campaigns to run\n", len(cs))
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// PostStaticCampaign inserts a campaign and all associated records into the database.
func PostStaticCampaign(c *StaticCampaign, uid int64) error {
	err := c.Validate()
	if err != nil {
		return err
	}
	// Fill in the details
	c.UserId = uid
	c.CreatedDate = time.Now().UTC()
	c.CompletedDate = time.Time{}
	c.Status = CampaignQueued
	if c.LaunchDate.IsZero() {
		c.LaunchDate = c.CreatedDate
	} else {
		c.LaunchDate = c.LaunchDate.UTC()
	}
	if c.LaunchDate.Before(c.CreatedDate) || c.LaunchDate.Equal(c.CreatedDate) {
		c.Status = CampaignInProgress
	}

	// Check to make sure the page exists
	p, err := GetPageByName(c.Page.Name, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"page": p.Name,
		}).Error("Page does not exist")
		return ErrPageNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}
	c.Page = p
	c.PageId = p.Id

	// Insert into the DB
	err = db.Save(c).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = c.AddEvent(&Event{Message: "Campaign Created"})
	if err != nil {
		log.Error(err)
	}

	err = db.Save(c).Error
	return err
}

//DeleteStaticCampaign deletes the specified campaign
func DeleteStaticCampaign(id int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Deleting campaign")
	// Delete all the campaign results
	err := db.Where("campaign_id=?", id).Delete(&Result{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&Event{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Delete the campaign
	err = db.Delete(&StaticCampaign{Id: id}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// CompleteStaticCampaign effectively "ends" a campaign.
// Any future emails clicked will return a simple "404" page.
func CompleteStaticCampaign(id int64, uid int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Marking campaign as complete")
	c, err := GetCampaign(id, uid)
	if err != nil {
		return err
	}

	// Don't overwrite original completed time
	if c.Status == CampaignComplete {
		return nil
	}
	// Mark the campaign as complete
	c.CompletedDate = time.Now().UTC()
	c.Status = CampaignComplete
	err = db.Where("id=? and user_id=?", id, uid).Save(&c).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
