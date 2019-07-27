
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE "static_campaigns" ("id" integer primary key autoincrement,"user_id" bigint,"name" varchar(255) NOT NULL,"created_date" datetime,"completed_date" datetime,"page_id" bigint,"status" varchar(255),"url" varchar(255),"launch_date" DATETIME);

ALTER TABLE "results" ADD static_campaign_id bigint;

ALTER TABLE "events" ADD static_campaign_id bigint;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE "static_campaigns"
DROP TABLE "results"
DROP TABLE "events"
