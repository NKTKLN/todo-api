package models

const (
	// Select
	SqlSelectUserById           = `SELECT * FROM "users" WHERE id = $1 LIMIT 1`
	SqlSelectUserByEmail        = `SELECT * FROM "users" WHERE email = $1 LIMIT 1`
	SqlSelectAllUsersById       = `SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id"`
	SqlSelectAllUsersByEmail    = `SELECT * FROM "users" WHERE email = $1 ORDER BY "users"."id"`
	SqlSelectAllUsersByUsername = `SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id"`

	SqlSelectListById                   = `SELECT * FROM "lists" WHERE id = $1 LIMIT 1`
	SqlSelectListByIdAndUserId          = `SELECT * FROM "lists" WHERE id = $1 AND user_id = $2 LIMIT 1`
	SqlSelectAllListsByUserId           = `SELECT * FROM "lists" WHERE user_id = $1 ORDER BY index`
	SqlSelectAllListsForEditIndex       = `SELECT * FROM "lists" WHERE user_id = $1 AND index > $2`
	SqlSelectMaxListIndex               = `SELECT max(index) FROM "lists" WHERE user_id = $1 LIMIT 1`
	SqlSelectAllListsToIncreaseTheIndex = `SELECT * FROM "lists" WHERE user_id = $1 AND index <= $2 AND index > $3`
	SqlSelectAllListsForIndexReduction  = `SELECT * FROM "lists" WHERE user_id = $1 AND index >= $2 AND index < $3`

	SqlSelectTaskById                   = `SELECT * FROM "tasks" WHERE id = $1 LIMIT 1`
	SqlSelectAllTasksByListId           = `SELECT * FROM "tasks" WHERE list_id = $1 ORDER BY index`
	SqlSelectAllTasksForEditIndex       = `SELECT * FROM "tasks" WHERE list_id = $1 AND index > $2`
	SqlSelectMaxTaskIndex               = `SELECT max(index) FROM "tasks" WHERE list_id = $1 LIMIT 1`
	SqlSelectAllTasksToIncreaseTheIndex = `SELECT * FROM "tasks" WHERE list_id = $1 AND index <= $2 AND index > $3`
	SqlSelectAllTasksForIndexReduction  = `SELECT * FROM "tasks" WHERE list_id = $1 AND index >= $2 AND index < $3`

	SqlSelectAllSubtasksByTaskId           = `SELECT * FROM "tasks" WHERE task_id = $1 ORDER BY index`
	SqlSelectTaskIdBySubtaskId             = `SELECT task_id FROM "tasks" WHERE id = $1 LIMIT 1`
	SqlSelectMaxSubtaskIndex               = `SELECT max(index) FROM "tasks" WHERE task_id = $1 LIMIT 1`
	SqlSelectAllSubtasksForEditIndex       = `SELECT * FROM "tasks" WHERE task_id = $1 AND index > $2`
	SqlSelectAllSubtasksToIncreaseTheIndex = `SELECT * FROM "tasks" WHERE task_id = $1 AND index <= $2 AND index > $3`
	SqlSelectAllSubtasksForIndexReduction  = `SELECT * FROM "tasks" WHERE task_id = $1 AND index >= $2 AND index < $3`

	// Select with join
	SqlSelectListIdWhereTask = `SELECT lists.id FROM "lists" INNER JOIN tasks ON lists.id=tasks.list_id WHERE user_id = $1 AND tasks.id = $2 LIMIT 1`

	// Insert
	SqlInsertUserData = `INSERT INTO "users" ("email","password","name","username","icon","id") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`

	SqlInsertListData = `INSERT INTO "lists" ("user_id","name","comment","index","id") VALUES ($1,$2,$3,$4,$5) RETURNING "id"`

	SqlInsertTaskData = `INSERT INTO "tasks" ("list_id","task_id","name","comment","index","categories","end_time","done","special","id") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING "id"`

	// Delete
	SqlDeleteUser = `DELETE FROM "users" WHERE "users"."id" = $1`

	SqlDeleteList = `DELETE FROM "lists" WHERE "lists"."id" = $1`

	SqlDeleteTask = `DELETE FROM "tasks" WHERE "tasks"."id" = $1`

	// TODO: replace Edit to Upload
	// Edit
	SqlEditUserName     = `UPDATE "users" SET "name"=$1 WHERE "users"."id" = $2`
	SqlEditUserIcon     = `UPDATE "users" SET "icon"=$1 WHERE id = $2`
	SqlEditUserEmail    = `UPDATE "users" SET "email"=$1 WHERE "users"."id" = $2`
	SqlEditUserUsername = `UPDATE "users" SET "username"=$1 WHERE "users"."id" = $2`
	SqlEditUserPassword = `UPDATE "users" SET "password"=$1 WHERE email = $2`

	SqlEditList      = `UPDATE "lists" SET "name"=$1,"comment"=$2 WHERE "id" = $3`
	SqlEditListIndex = `UPDATE "lists" SET "index"=$1 WHERE id = $2`

	SqlEditTask      = `UPDATE "tasks" SET "name"=$1,"comment"=$2,"categories"=$3,"end_time"=$4,"done"=$5,"special"=$6 WHERE "id" = $7`
	SqlEditTaskIndex = `UPDATE "tasks" SET "index"=$1 WHERE id = $2`
)
