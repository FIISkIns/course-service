package main

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
)

type BaseCourseInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AchievementInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Type        string `json:"type"`
}

type CourseInfo struct {
	BaseCourseInfo
	TaskGroups []TaskGroup `json:"taskGroups"`
}

type TaskGroup struct {
	Title string          `json:"title"`
	Tasks []*BaseTaskInfo `json:"tasks"`
}

type BaseTaskInfo struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

type TaskInfo struct {
	BaseTaskInfo `yaml:",inline"`
	Body string  `json:"body"`
}

var courseInfo CourseInfo
var achievementsInfo = make([]AchievementInfo, 0)
var cachedTasks = make(map[string]*TaskInfo)

func loadCourseYaml(filePath string, v interface{}) error {
	log.Println("Loading course file", filePath)

	data, err := ioutil.ReadFile(path.Join(config.Path, filePath))
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, v)
}

func parseTaskPath(p string) string {
	parts := strings.Split(p, ".")
	parts[len(parts)-1] += ".yml"
	return path.Join(parts...)
}

func loadTask(task string) (*TaskInfo, error) {
	if cachedTasks[task] != nil {
		return cachedTasks[task], nil
	}

	info := &TaskInfo{}
	err := loadCourseYaml(path.Join("tasks", parseTaskPath(task)), &info)
	if err != nil {
		return nil, err
	}
	info.Id = task

	cachedTasks[task] = info
	return info, nil
}

func loadCourseInfo() {
	type TaskGroupRaw struct {
		Title string
		Tasks []string
	}

	type AchievementsInfoRaw struct {
		AchievementInfo `yaml:",inline"`
	}

	type CourseInfoRaw struct {
		BaseCourseInfo                     `yaml:",inline"`
		TaskGroups   []TaskGroupRaw        `yaml:"task-groups"`
		Achievements []AchievementsInfoRaw `yaml:"achievements"`
	}

	var info CourseInfoRaw
	err := loadCourseYaml("course.yml", &info)
	if err != nil {
		log.Panicln("While loading course info", err)
	}

	courseInfo = CourseInfo{
		BaseCourseInfo: info.BaseCourseInfo,
		TaskGroups:     make([]TaskGroup, len(info.TaskGroups)),
	}

	var achievementInfo AchievementInfo
	for _, achievement := range info.Achievements {
		achievementInfo.Title = achievement.Title
		achievementInfo.Description = achievement.Description
		achievementInfo.Icon = achievement.Icon
		achievementInfo.Type = achievement.Type
		achievementsInfo = append(achievementsInfo, achievementInfo)
	}

	for i, group := range info.TaskGroups {
		courseInfo.TaskGroups[i] = TaskGroup{
			Title: group.Title,
			Tasks: make([]*BaseTaskInfo, len(group.Tasks)),
		}

		for j, taskPath := range group.Tasks {
			task, err := loadTask(taskPath)
			if err != nil {
				log.Panicln("While loading task info", taskPath, err)
			}
			courseInfo.TaskGroups[i].Tasks[j] = &task.BaseTaskInfo
		}
	}

	log.Println("Course info loaded successfully.")
	log.Println("Tasks loaded:", len(cachedTasks))
}

func HandleGetCourseInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data, err := json.Marshal(&courseInfo.BaseCourseInfo)
	if err != nil {
		http.Error(w, "Could not serialize course info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func HandleGetTasks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data, err := json.Marshal(&courseInfo.TaskGroups)
	if err != nil {
		http.Error(w, "Could not serialize task groups", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func HandleGetTaskInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	task, err := loadTask(ps.ByName("id"))
	if err != nil {
		log.Println("error while serving task info:", err)
		http.Error(w, "Could not load task", http.StatusNotFound)
		return
	}

	data, err := json.Marshal(task)
	if err != nil {
		log.Println("error while serving task info:", err)
		http.Error(w, "Could not serialize task info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func HandleGetAchievementsInfo(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	data, err := json.Marshal(&achievementsInfo)
	if err != nil {
		http.Error(w, "Could not serialize course info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func main() {
	initConfig()
	loadCourseInfo()

	router := httprouter.New()
	router.GET("/", HandleGetCourseInfo)
	router.GET("/tasks", HandleGetTasks)
	router.GET("/tasks/:id", HandleGetTaskInfo)
	router.GET("/achievements", HandleGetAchievementsInfo)
	router.ServeFiles("/static/*filepath", http.Dir(path.Join(config.Path, "resources")))

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), router))
}
