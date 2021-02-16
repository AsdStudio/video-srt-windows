package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/unknwon/i18n"
	"log"
	"runtime"
	"strings"
	"time"
	. "videosrt/app"
	"videosrt/app/ffmpeg"
	"videosrt/app/tool"
)

//应用版本号
const APP_VERSION = "0.3.2"

var AppRootDir string
var mw *MyMainWindow

var UseLanguage = Settings.GetCacheAppSettingsData().ShowLanguage
var Fontfamily string
var (
	outputSrtChecked *walk.CheckBox
	outputLrcChecked *walk.CheckBox
	outputTxtChecked *walk.CheckBox

	globalFilterChecked *walk.CheckBox
	definedFilterChecked *walk.CheckBox
)


func init()  {
	//设置可同时执行的最大CPU数
	runtime.GOMAXPROCS(runtime.NumCPU())
	//MY Window
	mw = new(MyMainWindow)

	AppRootDir = GetAppRootDir()
	switch UseLanguage {
	case "zh_Hans":
		Fontfamily="Microsoft YaHei"
	case "zh_Hant":
		Fontfamily="Microsoft JhengHei"
	default:
		Fontfamily="Tahoma"
	}

	_ = tool.LocaleInit(UseLanguage, fmt.Sprintf("language/%s.ini",UseLanguage))
	//log locale
	log.Println(UseLanguage,fmt.Sprintf("language/%s.ini",UseLanguage),i18n.Tr(UseLanguage,"Hello"))
	//log
	if AppRootDir == "" {
		panic("应用根目录获取失败")
	}

	//校验ffmpeg环境
	if e := ffmpeg.VailFfmpegLibrary(); e != nil {
		//尝试自动引入 ffmpeg 环境
		ffmpeg.VailTempFfmpegLibrary(AppRootDir)
	}
}

var locale string

func main() {
	var taskFiles = new(TaskHandleFile)

	var logText *walk.TextEdit

	var operateEngineDb *walk.DataBinder
	var operateTranslateEngineDb *walk.DataBinder
	var operateTranslateDb *walk.DataBinder
	var operateDb *walk.DataBinder
	var operateFilter *walk.DataBinder

	var operateFrom = new(OperateFrom)

	var startBtn *walk.PushButton //生成字幕Btn
	var startTranslateBtn *walk.PushButton //字幕翻译Btn
	var engineOptionsBox *walk.ComboBox
	var translateEngineOptionsBox *walk.ComboBox
	var dropFilesEdit *walk.TextEdit

	var appSettings = Settings.GetCacheAppSettingsData()
	var appFilter = Filter.GetCacheAppFilterData()

	//初始化展示配置
	operateFrom.Init(appSettings)

	//日志
	var tasklog = NewTasklog(logText)

	//字幕生成应用
	var videosrt = NewApp(AppRootDir)
	//注册日志事件
	videosrt.SetLogHandler(func(s string, video string) {
		baseName := tool.GetFileBaseName(video)
		strs := strings.Join([]string{"【" , baseName , "】" , s} , "")
		//追加日志
		tasklog.AppendLogText(strs)
	})
	//字幕输出目录
	videosrt.SetSrtDir(appSettings.SrtFileDir)
	//注册[字幕生成]多任务
	var multitask = NewVideoMultitask(appSettings.MaxConcurrency)


	//字幕翻译应用
	var srtTranslateApp = NewSrtTranslateApp(AppRootDir)
	//注册日志回调事件
	srtTranslateApp.SetLogHandler(func(s string, file string) {
		baseName := tool.GetFileBaseName(file)
		strs := strings.Join([]string{"【" , baseName , "】" , s} , "")
		//追加日志
		tasklog.AppendLogText(strs)
	})
	//文件输出目录
	srtTranslateApp.SetSrtDir(appSettings.SrtFileDir)
	//注册[字幕翻译]多任务
	var srtTranslateMultitask = NewTranslateMultitask(appSettings.MaxConcurrency)


	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Icon:"./data/img/index.png",
		Title:"VideoSrt - 一键字幕生成/字幕翻译神器" + " -v" + APP_VERSION,
		Font:Font{Family: Fontfamily, PointSize: 9},
		ToolBar: ToolBar{
			ButtonStyle: ToolBarButtonImageBeforeText,
			Items: []MenuItem{
				Menu{
					Text:"打开",
					Image: "./data/img/open.png",
					Items: []MenuItem{
						Action{
							Image:  "./data/img/media.png",
							Text:   "媒体文件",
							OnTriggered: func() {
								dlg := new(walk.FileDialog)
								//选择待操作的文件列表
								//dlg.FilePath = mw.prevFilePath
								dlg.Filter = "Media Files (*.mp4;*.mpeg;*.mkv;*.wmv;*.avi;*.m4v;*.mov;*.flv;*.rmvb;*.3gp;*.f4v;*.mp3;*.wav;*.aac;*.wma;*.flac;*.m4a;*.srt)|*.mp4;*.mpeg;*.mkv;*.wmv;*.avi;*.m4v;*.mov;*.flv;*.rmvb;*.3gp;*.f4v;*.mp3;*.wav;*.aac;*.wma;*.flac;*.m4a;*.srt"
								dlg.Title = "选择待操作的媒体文件"

								ok, err := dlg.ShowOpenMultiple(mw);
								if err != nil {
									mw.NewErrormationTips("错误" , err.Error())
									return
								}
								if ok == false {
									return
								}

								//校验文件数量
								if len(dlg.FilePaths) == 0 {
									return
								}

								//检测文件列表
								result , err := VaildateHandleFiles(dlg.FilePaths , true , true)
								if err != nil {
									mw.NewErrormationTips("错误" , err.Error())
									return
								}

								taskFiles.Files = result
								dropFilesEdit.SetText(strings.Join(result, "\r\n"))
							},
						},
					},
				},
				Menu{
					Text:  "新建",
					Image: "./data/img/new.png",
					Items: []MenuItem{
						Action{
							Image:  "./data/img/voice.png",
							Text:   "语音引擎（阿里云）",
							OnTriggered: func() {
								mw.RunSpeechEngineSettingDialog(mw , func() {
									thisData := Engine.GetEngineOptionsSelects()

									//校验选择的翻译引擎是否存在
									_ , ok := Engine.GetEngineById(appSettings.CurrentEngineId)
									if appSettings.CurrentEngineId == 0 || !ok {
										appSettings.CurrentEngineId = thisData[0].Id

										//更新缓存
										Settings.SetCacheAppSettingsData(appSettings)
									}

									//重新加载选项
									_ = engineOptionsBox.SetModel(thisData)
									//重置index
									engIndex := Engine.GetCurrentIndex(thisData , appSettings.CurrentEngineId)
									if engIndex != -1 {
										_ = engineOptionsBox.SetCurrentIndex(engIndex)
									}
									operateFrom.EngineId = appSettings.CurrentEngineId
								})
							},
						},
						Action{
							Image:  "./data/img/translate.png",
							Text:   "翻译引擎（百度翻译）",
							OnTriggered: func() {
								mw.RunBaiduTranslateEngineSettingDialog(mw , func() {
									thisData := Translate.GetTranslateEngineOptionsSelects()

									//校验选择的翻译引擎是否存在
									_ , ok := Engine.GetEngineById(appSettings.CurrentEngineId)
									if appSettings.CurrentTranslateEngineId == 0 || !ok {
										appSettings.CurrentTranslateEngineId = thisData[0].Id
										//更新缓存
										Settings.SetCacheAppSettingsData(appSettings)
									}

									//重新加载选项
									_ = translateEngineOptionsBox.SetModel(thisData)
									//重置index
									engIndex := Translate.GetCurrentTranslateEngineIndex(thisData , appSettings.CurrentTranslateEngineId)
									if engIndex != -1 {
										_ = translateEngineOptionsBox.SetCurrentIndex(engIndex)
									}
									operateFrom.TranslateEngineId = appSettings.CurrentTranslateEngineId
								})
							},
						},
						Action{
							Image:  "./data/img/translate.png",
							Text:   "翻译引擎（腾讯云）",
							OnTriggered: func() {
								mw.RunTengxunyunTranslateEngineSettingDialog(mw , func() {
									thisData := Translate.GetTranslateEngineOptionsSelects()
									if appSettings.CurrentTranslateEngineId == 0 {
										appSettings.CurrentTranslateEngineId = thisData[0].Id
										//更新缓存
										Settings.SetCacheAppSettingsData(appSettings)
									}

									//重新加载选项
									_ = translateEngineOptionsBox.SetModel(thisData)
									//重置index
									engIndex := Translate.GetCurrentTranslateEngineIndex(thisData , appSettings.CurrentTranslateEngineId)
									if engIndex != -1 {
										_ = translateEngineOptionsBox.SetCurrentIndex(engIndex)
									}
									operateFrom.TranslateEngineId = appSettings.CurrentTranslateEngineId
								})
							},
						},
					},
				},
				Menu{
					Text:  "设置",
					Image: "./data/img/Settings.png",
					Items: []MenuItem{
						Action{
							Text:    "OSS对象存储设置",
							Image:   "./data/img/oss.png",
							OnTriggered: func() {
								mw.RunObjectStorageSettingDialog(mw)
							},
						},
						Action{
							Text:    "软件设置",
							Image:   "./data/img/app-Settings.png",
							OnTriggered: func() {
								mw.RunAppSettingDialog(mw , func(Settings *AppSettings) {
									//更新配置
									appSettings.MaxConcurrency = Settings.MaxConcurrency
									appSettings.SrtFileDir = Settings.SrtFileDir
									appSettings.CloseNewVersionMessage = Settings.CloseNewVersionMessage
									appSettings.CloseAutoDeleteOssTempFile = Settings.CloseAutoDeleteOssTempFile
									appSettings.CloseIntelligentBlockSwitch = Settings.CloseIntelligentBlockSwitch
									appSettings.ShowLanguage = Settings.ShowLanguage

									multitask.SetMaxConcurrencyNumber( Settings.MaxConcurrency )
									srtTranslateMultitask.SetMaxConcurrencyNumber( Settings.MaxConcurrency )
								})
							},
						},
					},
				},
				Menu{
					Text:  "帮助文档/支持",
					Image: "./data/img/about.png",
					Items: []MenuItem{
						Action{
							Text:        "github",
							Image:      "./data/img/github.png",
							OnTriggered: mw.OpenAboutGithub,
						},
						Action{
							Text:        "gitee",
							Image:      "./data/img/gitee.png",
							OnTriggered: mw.OpenAboutGitee,
						},
						Action{
							Text:        "帮助文档",
							Image:      "./data/img/version.png",
							OnTriggered: func() {
								_ = tool.OpenUrl("https://www.yuque.com/viggo-t7cdi/videosrt")
							},
						},
						Action{
							Text:        "赞助/打赏",
							OnTriggered: func() {
								_ = tool.OpenUrl("https://gitee.com/641453620/video-srt-windows#%E6%8D%90%E8%B5%A0%E6%94%AF%E6%8C%81")
							},
						},
						Action{
							Text:        "QQ交流群",
							Checked:false,
							Visible:false,
							Checkable:false,
							OnTriggered: func() {
								_ = tool.OpenUrl("https://gitee.com/641453620/video-srt-windows#%E4%BA%A4%E6%B5%81%E8%81%94%E7%B3%BB")
							},
						},
					},
				},
				Menu{
					Text:  "语音合成配音/文章转视频",
					Image: "./data/img/muyan.png",
					OnTriggered: func() {
						_ = tool.OpenUrl("https://www.mu-yan.net/")
					},
				},
			},
		},
		Size: Size{800, 650},
		MinSize: Size{300, 650},
		Layout:  VBox{},
		Children: []Widget{
			HSplitter{
				Children: []Widget{
					Composite{
						MinSize:Size{Height:31,Width:400},
						DataBinder: DataBinder{
							AssignTo:    &operateEngineDb,
							DataSource:   operateFrom,
						},
						Layout: Grid{Columns: 3},
						Children: []Widget{
							Label{
								Text: "语音引擎：",
							},
							ComboBox{
								AssignTo:&engineOptionsBox,
								Value: Bind("EngineId", SelRequired{}),
								BindingMember: "Id",
								DisplayMember: "Name",
								Model:  Engine.GetEngineOptionsSelects(),
								OnCurrentIndexChanged: func() {
									_ = operateEngineDb.Submit()

									if operateFrom.EngineId == 0 {
										return
									}

									appSettings.CurrentEngineId = operateFrom.EngineId
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							PushButton{
								Text: "删除",
								MaxSize:Size{50 , 55},
								OnClicked: func() {
									var thisEngineOptions = make([]*EngineSelects , 0)
									thisEngineOptions = Engine.GetEngineOptionsSelects()
									//删除校验
									if appSettings.CurrentEngineId == 0 {
										mw.NewErrormationTips("错误" , "请选择要操作的语音引擎")
										return
									}
									if len(thisEngineOptions) <= 1 {
										mw.NewErrormationTips("错误" , "不能删除全部语音引擎")
										return
									}

									//删除引擎
									if ok := Engine.RemoveCacheAliyunEngineData(appSettings.CurrentEngineId);ok == false {
										//删除失败
										mw.NewErrormationTips("错误" , "语音引擎删除失败")
										return
									}

									thisEngineOptions = Engine.GetEngineOptionsSelects()

									//重新加载列表
									_ = engineOptionsBox.SetModel(thisEngineOptions)

									appSettings.CurrentEngineId = thisEngineOptions[0].Id
									operateFrom.EngineId = appSettings.CurrentEngineId
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
									//更新下标
									_ = engineOptionsBox.SetCurrentIndex(0)
								},
							},
						},
					},

					Composite{
						MinSize:Size{Height:31,Width:400},
						DataBinder: DataBinder{
							AssignTo:    &operateTranslateEngineDb,
							DataSource:   operateFrom,
						},
						Layout: Grid{Columns: 3},
						Children: []Widget{
							Label{
								Text: "翻译引擎：",
							},
							ComboBox{
								AssignTo:&translateEngineOptionsBox,
								Value: Bind("TranslateEngineId", SelRequired{}),
								BindingMember: "Id",
								DisplayMember: "Name",
								Model:  Translate.GetTranslateEngineOptionsSelects(),
								OnCurrentIndexChanged: func() {
									_ = operateTranslateEngineDb.Submit()

									if operateFrom.TranslateEngineId == 0 {
										return
									}

									appSettings.CurrentTranslateEngineId = operateFrom.TranslateEngineId
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							PushButton{
								Text: "删除",
								MaxSize:Size{50 , 55},
								OnClicked: func() {
									var thisEngineOptions = make([]*TranslateEngineSelects , 0)
									thisEngineOptions = Translate.GetTranslateEngineOptionsSelects()
									//删除校验
									if appSettings.CurrentTranslateEngineId == 0 {
										mw.NewErrormationTips("错误" , "请选择要删除的翻译引擎")
										return
									}
									if len(thisEngineOptions) <= 1 {
										mw.NewErrormationTips("错误" , "不能删除全部翻译引擎")
										return
									}

									//删除引擎
									if ok := Translate.RemoveCacheTranslateEngineData(appSettings.CurrentTranslateEngineId);ok == false {
										//删除失败
										mw.NewErrormationTips("错误" , "翻译引擎删除失败")
										return
									}

									thisEngineOptions = Translate.GetTranslateEngineOptionsSelects()

									//重新加载列表
									_ = translateEngineOptionsBox.SetModel(thisEngineOptions)

									appSettings.CurrentTranslateEngineId = thisEngineOptions[0].Id
									operateFrom.TranslateEngineId = appSettings.CurrentTranslateEngineId
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
									//更新下标
									_ = translateEngineOptionsBox.SetCurrentIndex(0)
								},
							},
						},
					},
				},
			},


			/*翻译设置*/
			HSplitter{
				Children:[]Widget{
					Composite{
						DataBinder: DataBinder{
							AssignTo:    &operateTranslateDb,
							DataSource:   operateFrom,
						},
						Layout: Grid{Columns: 4},
						Children: []Widget{
							Label{
								Text: "翻译设置：",
							},
							CheckBox{
								Text:"开启翻译",
								Checked: Bind("TranslateSwitch"),
								OnClicked: func() {
									_ = operateTranslateDb.Submit()

									appSettings.TranslateSwitch = operateFrom.TranslateSwitch
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							CheckBox{
								Text:"双语字幕",
								Checked: Bind("BilingualSubtitleSwitch"),
								OnClicked: func() {
									_ = operateTranslateDb.Submit()

									appSettings.BilingualSubtitleSwitch = operateFrom.BilingualSubtitleSwitch
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							CheckBox{
								Text:"主字幕（输入语言）",
								Checked: Bind("OutputMainSubtitleInputLanguage"),
								OnClicked: func() {
									_ = operateTranslateDb.Submit()

									appSettings.OutputMainSubtitleInputLanguage = operateFrom.OutputMainSubtitleInputLanguage
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},

							//输入语言
							Label{
								Text: "输入语言：",
							},
							ComboBox{
								Value: Bind("InputLanguage", SelRequired{}),
								BindingMember: "Id",
								DisplayMember: "Name",
								Model: GetTranslateInputLanguageOptionsSelects(),
								ColumnSpan: 3,
								MaxSize:Size{Width:80},
								OnCurrentIndexChanged: func() {
									_ = operateTranslateDb.Submit()
									appSettings.InputLanguage = operateFrom.InputLanguage
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							//输出语言
							Label{
								Text: "输出语言：",
							},
							ComboBox{
								Value: Bind("OutputLanguage", SelRequired{}),
								BindingMember: "Id",
								DisplayMember: "Name",
								Model: GetTranslateOutputLanguageOptionsSelects(),
								ColumnSpan: 3,
								MaxSize:Size{Width:80},
								OnCurrentIndexChanged: func() {
									_ = operateTranslateDb.Submit()
									appSettings.OutputLanguage = operateFrom.OutputLanguage
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
						},
					},
				},
			},


			/*过滤器设置*/
			HSplitter{
				Children:[]Widget{
					Composite{
						DataBinder: DataBinder{
							AssignTo:    &operateFilter,
							DataSource:   appFilter,
						},
						Layout: Grid{Columns: 5},
						Children: []Widget{
							Label{
								Text: "过滤设置：",
							},
							CheckBox{
								AssignTo:&globalFilterChecked,
								Text:"语气词过滤 ",
								Checked: Bind("GlobalFilter.Switch"),
								OnClicked: func() {
									_ = operateFilter.Submit()
									//更新缓存
									Filter.SetCacheAppFilterData(appFilter)
								},
							},
							CheckBox{
								AssignTo:&definedFilterChecked,
								Text:"自定义过滤  ",
								Checked: Bind("DefinedFilter.Switch"),
								OnClicked: func() {
									_ = operateFilter.Submit()
									//更新缓存
									Filter.SetCacheAppFilterData(appFilter)
								},
							},

							PushButton{
								Text: "语气词过滤设置",
								MaxSize:Size{95 , 55},
								OnClicked: func() {
									mw.RunGlobalFilterSettingDialog(mw , appFilter.GlobalFilter.Words , func(words string) {
										appFilter.GlobalFilter.Words = words
										//更新缓存
										Filter.SetCacheAppFilterData(appFilter)
									})
								},
							},
							PushButton{
								Text: "自定义过滤设置",
								MaxSize:Size{95 , 55},
								OnClicked: func() {
									mw.RunDefinedFilterSettingDialog(mw , appFilter.DefinedFilter.Rule , func(rule []*AppDefinedFilterRule) {
										appFilter.DefinedFilter.Rule = rule
										//更新缓存
										Filter.SetCacheAppFilterData(appFilter)
									})
								},
							},
						},
					},
				},
			},


			/*输出设置*/
			HSplitter{
				Children:[]Widget{
					Composite{
						DataBinder: DataBinder{
							AssignTo:    &operateDb,
							DataSource:   operateFrom,
						},
						Layout: Grid{Columns: 4},
						Children: []Widget{
							Label{
								Text: "输出文件：",
							},
							CheckBox{
								AssignTo:&outputSrtChecked,
								Text:"SRT文件",
								Checked: Bind("OutputSrt"),
								OnClicked: func() {
									_ = operateDb.Submit()

									operateFrom.OutputType.SRT = operateFrom.OutputSrt
									appSettings.OutputType = operateFrom.OutputType
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							CheckBox{
								AssignTo:&outputLrcChecked,
								Text:"LRC文件",
								Checked: Bind("OutputLrc"),
								OnClicked: func() {
									_ = operateDb.Submit()

									operateFrom.OutputType.LRC = operateFrom.OutputLrc
									appSettings.OutputType = operateFrom.OutputType
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							CheckBox{
								AssignTo:&outputTxtChecked,
								Text:"普通文本",
								Checked: Bind("OutputTxt"),
								OnClicked: func() {
									_ = operateDb.Submit()

									operateFrom.OutputType.TXT = operateFrom.OutputTxt
									appSettings.OutputType = operateFrom.OutputType
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							//输出文件编码
							Label{
								Text: "输出编码：",
							},
							ComboBox{
								Value: Bind("OutputEncode", SelRequired{}),
								BindingMember: "Id",
								DisplayMember: "Name",
								Model: GetOutputEncodeOptionsSelects(),
								ColumnSpan: 3,
								MaxSize:Size{Width:80},
								OnCurrentIndexChanged: func() {
									_ = operateDb.Submit()
									appSettings.OutputEncode = operateFrom.OutputEncode
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
							//输出文件音轨
							Label{
								Text: "输出音轨：",
							},
							ComboBox{
								Value: Bind("SoundTrack", SelRequired{}),
								BindingMember: "Id",
								DisplayMember: "Name",
								Model: GetSoundTrackSelects(),
								ColumnSpan: 3,
								MaxSize:Size{Width:60},
								OnCurrentIndexChanged: func() {
									_ = operateDb.Submit()
									appSettings.SoundTrack = operateFrom.SoundTrack
									//更新缓存
									Settings.SetCacheAppSettingsData(appSettings)
								},
							},
						},
					},
				},
			},

			HSplitter{
				Children: []Widget{
					TextEdit{
						AssignTo: &dropFilesEdit,
						ReadOnly: true,
						Text:     "将需要处理的媒体文件，拖入放到这里\r\n\r\n支持的视频格式：.mp4 , .mpeg , .mkv , .wmv , .avi , .m4v , .mov , .flv , .rmvb , .3gp , .f4v\r\n支持的音频格式：.mp3 , .wav , .aac , .wma , .flac , .m4a\r\n支持的字幕格式：.srt",
						TextColor:walk.RGB(136 , 136 , 136),
						VScroll:true,
					},
					TextEdit{
						AssignTo: &logText,
						ReadOnly: true,
						Text:"这里是日志输出区",
						TextColor:walk.RGB(136 , 136 , 136),
						//HScroll:true,
						VScroll:true,
					},
				},
			},
			HSplitter{
				Children: []Widget{
					PushButton{
						AssignTo: &startBtn,
						Text: "生成字幕",
						MinSize:Size{Height:50},
						OnClicked: func() {

							tlens := len(taskFiles.Files)
							if tlens == 0 {
								//兼容外部调用
								tempDropFilesEdit := dropFilesEdit.Text()
								if tempDropFilesEdit != "" {
									tempFileLists := strings.Split(tempDropFilesEdit , "\r\n")
									//检测文件列表
									tempResult , _ := VaildateHandleFiles(tempFileLists , true ,false)
									if len(tempResult) != 0 {
										taskFiles.Files = tempResult
										dropFilesEdit.SetText(strings.Join(tempResult, "\r\n"))
									}
								}

								if len(taskFiles.Files) == 0 {
									mw.NewErrormationTips("错误" , "请先拖入要处理的媒体文件")
									return
								}
							}

							//校验文件列表
							if _,e := VaildateHandleFiles(taskFiles.Files , true , false); e!=nil {
								mw.NewErrormationTips("错误" , e.Error())
								return
							}
							//设置随机种子
							tool.SetRandomSeed()

							//查询应用配置
							tempAppSetting := Settings.GetCacheAppSettingsData()

							//参数校验
							if !operateFrom.OutputType.SRT && !operateFrom.OutputType.LRC && !operateFrom.OutputType.TXT {
								mw.NewErrormationTips("错误" , "至少选择一种输出文件")
								return
							}
							ossData := Oss.GetCacheAliyunOssData()
							if ossData.Endpoint == "" {
								mw.NewErrormationTips("错误" , "请先设置Oss对象配置")
								return
							}
							//校验输入语言
							if tempAppSetting.InputLanguage != LANGUAGE_ZH &&
								tempAppSetting.InputLanguage != LANGUAGE_EN &&
								tempAppSetting.InputLanguage != LANGUAGE_JP &&
								tempAppSetting.InputLanguage != LANGUAGE_KOR &&
								tempAppSetting.InputLanguage != LANGUAGE_RU &&
								tempAppSetting.InputLanguage != LANGUAGE_SPA {
								mw.NewErrormationTips("错误" , "由于语音提供商的限制，生成字幕仅支持中文、英文、日语、韩语、俄语、西班牙语")
								return
							}

							//查询选择的语音引擎
							if tempAppSetting.CurrentEngineId == 0 {
								mw.NewErrormationTips("错误" , "请先新建/选择语音引擎")
								return
							}
							currentEngine , ok := Engine.GetEngineById(tempAppSetting.CurrentEngineId)
							if !ok {
								mw.NewErrormationTips("错误" , "你选择的语音引擎不存在")
								return
							}

							//翻译配置
							tempTranslateCfg := new(VideoSrtTranslateStruct)
							tempTranslateCfg.TranslateSwitch = tempAppSetting.TranslateSwitch
							tempTranslateCfg.BilingualSubtitleSwitch = tempAppSetting.BilingualSubtitleSwitch
							tempTranslateCfg.InputLanguage = tempAppSetting.InputLanguage
							tempTranslateCfg.OutputLanguage = tempAppSetting.OutputLanguage
							tempTranslateCfg.OutputMainSubtitleInputLanguage = tempAppSetting.OutputMainSubtitleInputLanguage

							if tempTranslateCfg.TranslateSwitch {
								//校验选择的翻译引擎
								if tempAppSetting.CurrentTranslateEngineId == 0 {
									mw.NewErrormationTips("错误" , "你开启了翻译功能，请先新建/选择翻译引擎")
									return
								}
								currentTranslateEngine , ok := Translate.GetTranslateEngineById(tempAppSetting.CurrentTranslateEngineId)
								if !ok {
									mw.NewErrormationTips("错误" , "你选择的翻译引擎不存在")
									return
								}
								if currentTranslateEngine.Supplier == TRANSLATE_SUPPLIER_BAIDU {
									tempTranslateCfg.BaiduTranslate = currentTranslateEngine.BaiduEngine
								}
								if currentTranslateEngine.Supplier == TRANSLATE_SUPPLIER_TENGXUNYUN {
									tempTranslateCfg.TengxunyunTranslate = currentTranslateEngine.TengxunyunEngine
								}
								tempTranslateCfg.Supplier = currentTranslateEngine.Supplier //设置翻译供应商
							}

							//加载配置
							videosrt.InitAppConfig(ossData , currentEngine)
							videosrt.InitTranslateConfig(tempTranslateCfg)
							videosrt.InitFilterConfig(appFilter)
							videosrt.SetSrtDir(appSettings.SrtFileDir)
							videosrt.SetSoundTrack(appSettings.SoundTrack)
							videosrt.SetMaxConcurrency(appSettings.MaxConcurrency)
							videosrt.SetCloseAutoDeleteOssTempFile(appSettings.CloseAutoDeleteOssTempFile)
							videosrt.SetCloseIntelligentBlockSwitch(appSettings.CloseIntelligentBlockSwitch)

							//设置输出文件
							videosrt.SetOutputType(operateFrom.OutputType)
							//输出编码
							if appSettings.OutputEncode != 0 {
								videosrt.SetOutputEncode(appSettings.OutputEncode)
							}

							multitask.SetVideoSrt(videosrt)
							//设置队列
							multitask.SetQueueFile(taskFiles.Files)

							var finish = false

							startBtn.SetEnabled(false)
							startTranslateBtn.SetEnabled(false)
							startBtn.SetText("任务运行中，请勿关闭软件窗口...")
							//清除log
							tasklog.ClearLogText()
							tasklog.AppendLogText("任务开始... \r\n")

							//运行
							multitask.Run()

							//回调链式执行
							videosrt.SetFailHandler(func(video string) {
								//运行下一任务
								multitask.RunOver()

								//任务完成
								if ok := multitask.FinishTask(); ok && finish == false {
									//延迟结束
									go func() {
										time.Sleep(time.Second)
										finish = true
										startBtn.SetEnabled(true)
										startTranslateBtn.SetEnabled(true)
										startBtn.SetText("生成字幕")

										tasklog.AppendLogText("\r\n\r\n任务完成！")

										//清空临时目录
										videosrt.ClearTempDir()
									}()
								}
							})
							videosrt.SetSuccessHandler(func(video string) {
								//运行下一任务
								multitask.RunOver()

								//任务完成
								if ok := multitask.FinishTask(); ok && finish == false {
									//延迟结束
									go func() {
										time.Sleep(time.Second)
										finish = true
										startBtn.SetEnabled(true)
										startTranslateBtn.SetEnabled(true)
										startBtn.SetText("生成字幕")

										tasklog.AppendLogText("\r\n\r\n任务完成！")

										//清空临时目录
										videosrt.ClearTempDir()
									}()
								}
							})

						},
					},

					PushButton{
						AssignTo: &startTranslateBtn,
						Text: "字幕处理",
						MinSize:Size{Height:50},
						OnClicked: func() {
							//待处理的文件
							tlens := len(taskFiles.Files)
							if tlens == 0 {
								mw.NewErrormationTips("错误" , "请先拖入需要处理的SRT字幕文件")
								return
							}
							//校验文件列表
							if _,e := VaildateHandleFiles(taskFiles.Files , false , true); e!=nil {
								mw.NewErrormationTips("错误" , e.Error())
								return
							}

							//设置随机种子
							tool.SetRandomSeed()

							//查询应用配置
							tempAppSetting := Settings.GetCacheAppSettingsData()

							//参数校验
							if !operateFrom.OutputType.SRT && !operateFrom.OutputType.LRC && !operateFrom.OutputType.TXT {
								mw.NewErrormationTips("错误" , "至少选择一种输出文件")
								return
							}

							//翻译配置
							tempTranslateCfg := new(SrtTranslateStruct)
							tempTranslateCfg.TranslateSwitch = tempAppSetting.TranslateSwitch
							tempTranslateCfg.BilingualSubtitleSwitch = tempAppSetting.BilingualSubtitleSwitch
							tempTranslateCfg.InputLanguage = tempAppSetting.InputLanguage
							tempTranslateCfg.OutputLanguage = tempAppSetting.OutputLanguage
							tempTranslateCfg.OutputMainSubtitleInputLanguage = tempAppSetting.OutputMainSubtitleInputLanguage

							if tempTranslateCfg.TranslateSwitch {
								//校验选择的翻译引擎
								if tempAppSetting.CurrentTranslateEngineId == 0 {
									mw.NewErrormationTips("错误" , "你开启了翻译功能，请先新建/选择翻译引擎")
									return
								}
								currentTranslateEngine , ok := Translate.GetTranslateEngineById(tempAppSetting.CurrentTranslateEngineId)
								if !ok {
									mw.NewErrormationTips("错误" , "你选择的翻译引擎不存在")
									return
								}
								if currentTranslateEngine.Supplier == TRANSLATE_SUPPLIER_BAIDU {
									tempTranslateCfg.BaiduTranslate = currentTranslateEngine.BaiduEngine
								}
								if currentTranslateEngine.Supplier == TRANSLATE_SUPPLIER_TENGXUNYUN {
									tempTranslateCfg.TengxunyunTranslate = currentTranslateEngine.TengxunyunEngine
								}
								tempTranslateCfg.Supplier = currentTranslateEngine.Supplier //设置翻译供应商
							}

							//加载配置
							srtTranslateApp.InitTranslateConfig(tempTranslateCfg)
							srtTranslateApp.InitFilterConfig(appFilter)
							srtTranslateApp.SetSrtDir(appSettings.SrtFileDir)
							srtTranslateApp.SetMaxConcurrency(appSettings.MaxConcurrency)

							//设置输出文件
							srtTranslateApp.SetOutputType(operateFrom.OutputType)
							//输出编码
							if appSettings.OutputEncode != 0 {
								srtTranslateApp.SetOutputEncode(appSettings.OutputEncode)
							}

							//队列设置
							srtTranslateMultitask.SetSrtTranslateApp(srtTranslateApp)
							srtTranslateMultitask.SetQueueFile(taskFiles.Files)

							var finish = false

							startBtn.SetEnabled(false)
							startTranslateBtn.SetEnabled(false)
							startTranslateBtn.SetText("任务运行中，请勿关闭软件窗口...")
							//清除log
							tasklog.ClearLogText()
							tasklog.AppendLogText("任务开始... \r\n")

							//运行
							srtTranslateMultitask.Run()

							//注册回调链式执行
							srtTranslateApp.SetFailHandler(func(file string) {
								//运行下一任务
								srtTranslateMultitask.RunOver()

								//任务完成
								if ok := srtTranslateMultitask.FinishTask(); ok && finish == false {
									//延迟结束
									go func() {
										time.Sleep(time.Second)
										finish = true
										startBtn.SetEnabled(true)
										startTranslateBtn.SetEnabled(true)
										startTranslateBtn.SetText("字幕翻译")

										tasklog.AppendLogText("\r\n\r\n任务完成！")
									}()
								}
							})
							srtTranslateApp.SetSuccessHandler(func(video string) {
								//运行下一任务
								srtTranslateMultitask.RunOver()

								//任务完成
								if ok := srtTranslateMultitask.FinishTask(); ok && finish == false {
									//延迟结束
									go func() {
										time.Sleep(time.Second)
										finish = true
										startBtn.SetEnabled(true)
										startTranslateBtn.SetEnabled(true)
										startTranslateBtn.SetText("字幕翻译")

										tasklog.AppendLogText("\r\n\r\n任务完成！")
									}()
								}
							})
						},
					},
				},
			},
		},
		OnDropFiles: func(files []string) {
			//检测文件列表
			result , err := VaildateHandleFiles(files , true ,true)
			if err != nil {
				mw.NewErrormationTips("错误" , err.Error())
				return
			}

			taskFiles.Files = result
			dropFilesEdit.SetText(strings.Join(result, "\r\n"))
		},
	}.Create()); err != nil {
		log.Fatal(err)

		time.Sleep(1 * time.Second)
	}

	//更新
	tasklog.SetTextEdit(logText)

	//校验依赖库
	if e := ffmpeg.VailFfmpegLibrary(); e != nil {
		mw.NewErrormationTips("错误" , "请先下载并安装 ffmpeg 软件，才可以正常使用软件哦")
		if locale != "zh_CN" {
			tool.OpenUrl("https://github.com/wxbool/video-srt-windows")
		}else {
			tool.OpenUrl("https://gitee.com/641453620/video-srt-windows")
		}
		return
	}

	//尝试校验新版本
	if appSettings.CloseNewVersionMessage == false {
		go func() {
			appV := new(AppVersion)
			if vtag, e := appV.GetVersion(); e == nil {
				if vtag != "" && tool.CompareVersion(vtag , APP_VERSION) == 1 {
					_ = appV.ShowVersionNotifyInfo(vtag , mw)
				}
			}
		}()
	}

	mw.Run()
}
