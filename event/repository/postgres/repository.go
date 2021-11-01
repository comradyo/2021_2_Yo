package postgres

import (
	error2 "backend/event/error"
	log "backend/logger"
	"backend/models"
	sql2 "database/sql"
	sql "github.com/jmoiron/sqlx"
	"strconv"
)

const logMessage = "event:repository:postgres:"

type Repository struct {
	db *sql.DB
}

func NewRepository(database *sql.DB) *Repository {
	return &Repository{
		db: database,
	}
}

const (
	checkAuthorQuery = `select author_id from "event" where id = $1`
	listQuery        = `select * from "event"`
	getEventQuery    = `select * from "event" where id = $1`
	createEventQuery = `insert into "event" 
		(title, description, text, city, category, viewed, img_url, date, geo, tag, author_id) 
		values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::varchar[], $11) 
		returning id`
	updateEventQuery = `update "event" set
		title = $1, description = $2, text = $3, city = $4, category = $5, 
		viewed = $6, img_url = $7, date = $8, geo = $9, tag = $10 
		where event.id = $11`
	deleteEventQuery = `delete from "event" where id = $1`
)

func (s *Repository) checkAuthor(eventId int, userId int) error {
	var authorId int
	query := checkAuthorQuery
	err := s.db.QueryRow(query, eventId).Scan(&authorId)
	if err != nil {
		return error2.ErrPostgres
	}
	if authorId == userId {
		return nil
	} else {
		return error2.ErrNotAllowed
	}
}

//TODO: Посмотреть, что делает sliceScan

func (s *Repository) List() ([]*models.Event, error) {
	query := listQuery
	rows, err := s.db.Queryx(query)
	if err != nil {
		if err == sql2.ErrNoRows {
			return nil, error2.ErrNoRows
		}
		return nil, error2.ErrPostgres
	}
	defer rows.Close()
	var resultEvents []*models.Event
	for rows.Next() {
		var e Event
		err := rows.StructScan(&e)
		if err != nil {
			return nil, error2.ErrPostgres
		}
		modelEvent := toModelEvent(&e)
		resultEvents = append(resultEvents, modelEvent)
	}
	return resultEvents, nil
}

func (s *Repository) GetEvent(eventId string) (*models.Event, error) {
	eventIdInt, err := strconv.Atoi(eventId)
	if err != nil {
		return nil, error2.ErrAtoi
	}
	query := getEventQuery
	var e Event
	err = s.db.Get(&e, query, eventIdInt)
	if err != nil {
		if err == sql2.ErrNoRows {
			return nil, error2.ErrNoRows
		}
		return nil, error2.ErrPostgres
	}
	var resultEvent *models.Event
	resultEvent = toModelEvent(&e)
	return resultEvent, nil
}

func (s *Repository) CreateEvent(e *models.Event) (string, error) {
	newEvent, err := toPostgresEvent(e)
	log.Debug("newEvent tags = ", newEvent.Tag)
	if err != nil {
		return "", err
	}
	var eventId int
	query := createEventQuery
	err = s.db.QueryRow(query,
		newEvent.Title,
		newEvent.Description,
		newEvent.Text,
		newEvent.City,
		newEvent.Category,
		newEvent.Viewed,
		newEvent.ImgUrl,
		newEvent.Date,
		newEvent.Geo,
		newEvent.Tag,
		newEvent.AuthorID).Scan(&eventId)
	if err != nil {
		log.Debug("Postgres:CreateEvent err =", err)
		if err == sql2.ErrNoRows {
			return "", error2.ErrNoRows
		}
		return "", error2.ErrPostgres
	}
	return strconv.Itoa(eventId), nil
}

func (s *Repository) UpdateEvent(updatedEvent *models.Event, userId string) error {
	eventIdInt, err := strconv.Atoi(updatedEvent.ID)
	if err != nil {
		return error2.ErrAtoi
	}
	userIdInt, err := strconv.Atoi(userId)
	if err != nil {
		return error2.ErrAtoi
	}
	err = s.checkAuthor(eventIdInt, userIdInt)
	if err != nil {
		return err
	}
	e, err := toPostgresEvent(updatedEvent)
	if err != nil {
		return err
	}
	e.ID = eventIdInt
	query := updateEventQuery
	_, err = s.db.Exec(query,
		e.Title,
		e.Description,
		e.Text,
		e.City,
		e.Category,
		e.Viewed,
		e.ImgUrl,
		e.Date,
		e.Geo,
		e.Tag,
		e.ID)
	if err != nil {
		return error2.ErrPostgres
	}
	return nil
}

func (s *Repository) DeleteEvent(eventId string, userId string) error {
	eventIdInt, err := strconv.Atoi(eventId)
	if err != nil {
		return error2.ErrAtoi
	}
	userIdInt, err := strconv.Atoi(userId)
	if err != nil {
		return error2.ErrAtoi
	}
	err = s.checkAuthor(eventIdInt, userIdInt)
	if err != nil {
		return err
	}
	query := deleteEventQuery
	_, err = s.db.Exec(query, eventIdInt)
	if err != nil {
		return error2.ErrPostgres
	}
	return nil
}
