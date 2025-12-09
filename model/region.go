package model

// Area 表示一个地区（如 "JP13" = "東京"）
type Area struct {
	ID   string // 例如 "JP13"
	Name string // 例如 "東京"
}

// Region 表示一个大区（如 "関東"）
type Region struct {
	ID    string // 例如 "kanto"
	Name  string // 例如 "関東"
	Areas []Area // 该大区下的所有地区
}

// AllRegions 所有大区列表
var AllRegions = []Region{
	{
		ID:   "hokkaido-tohoku",
		Name: "北海道・東北",
		Areas: []Area{
			{ID: "JP1", Name: "北海道"},
			{ID: "JP2", Name: "青森"},
			{ID: "JP3", Name: "岩手"},
			{ID: "JP4", Name: "宮城"},
			{ID: "JP5", Name: "秋田"},
			{ID: "JP6", Name: "山形"},
			{ID: "JP7", Name: "福島"},
		},
	},
	{
		ID:   "kanto",
		Name: "関東",
		Areas: []Area{
			{ID: "JP8", Name: "茨城"},
			{ID: "JP9", Name: "栃木"},
			{ID: "JP10", Name: "群馬"},
			{ID: "JP11", Name: "埼玉"},
			{ID: "JP12", Name: "千葉"},
			{ID: "JP13", Name: "東京"},
			{ID: "JP14", Name: "神奈川"},
		},
	},
	{
		ID:   "hokuriku-koushinetsu",
		Name: "北陸・甲信越",
		Areas: []Area{
			{ID: "JP15", Name: "新潟"},
			{ID: "JP19", Name: "山梨"},
			{ID: "JP20", Name: "長野"},
			{ID: "JP17", Name: "石川"},
			{ID: "JP16", Name: "富山"},
			{ID: "JP18", Name: "福井"},
		},
	},
	{
		ID:   "chubu",
		Name: "中部",
		Areas: []Area{
			{ID: "JP23", Name: "愛知"},
			{ID: "JP21", Name: "岐阜"},
			{ID: "JP22", Name: "静岡"},
			{ID: "JP24", Name: "三重"},
		},
	},
	{
		ID:   "kinki",
		Name: "近畿",
		Areas: []Area{
			{ID: "JP27", Name: "大阪"},
			{ID: "JP28", Name: "兵庫"},
			{ID: "JP26", Name: "京都"},
			{ID: "JP25", Name: "滋賀"},
			{ID: "JP29", Name: "奈良"},
			{ID: "JP30", Name: "和歌山"},
		},
	},
	{
		ID:   "chugoku-shikoku",
		Name: "中国・四国",
		Areas: []Area{
			{ID: "JP33", Name: "岡山"},
			{ID: "JP34", Name: "広島"},
			{ID: "JP31", Name: "鳥取"},
			{ID: "JP32", Name: "島根"},
			{ID: "JP35", Name: "山口"},
			{ID: "JP37", Name: "香川"},
			{ID: "JP36", Name: "徳島"},
			{ID: "JP38", Name: "愛媛"},
			{ID: "JP39", Name: "高知"},
		},
	},
	{
		ID:   "kyushu",
		Name: "九州・沖縄",
		Areas: []Area{
			{ID: "JP40", Name: "福岡"},
			{ID: "JP41", Name: "佐賀"},
			{ID: "JP42", Name: "長崎"},
			{ID: "JP43", Name: "熊本"},
			{ID: "JP44", Name: "大分"},
			{ID: "JP45", Name: "宮崎"},
			{ID: "JP46", Name: "鹿児島"},
			{ID: "JP47", Name: "沖縄"},
		},
	},
}

// AllAreas 获取所有地区的平铺列表
func AllAreas() []Area {
	var areas []Area
	for _, region := range AllRegions {
		areas = append(areas, region.Areas...)
	}
	return areas
}

// FindAreaByID 根据 ID 查找地区
func FindAreaByID(areaID string) *Area {
	for _, region := range AllRegions {
		for _, area := range region.Areas {
			if area.ID == areaID {
				return &area
			}
		}
	}
	return nil
}

// FindRegionByAreaID 根据地区 ID 查找所属大区
func FindRegionByAreaID(areaID string) *Region {
	for i, region := range AllRegions {
		for _, area := range region.Areas {
			if area.ID == areaID {
				return &AllRegions[i]
			}
		}
	}
	return nil
}

// GetAreaIndex 获取地区在平铺列表中的索引
func GetAreaIndex(areaID string) int {
	areas := AllAreas()
	for i, area := range areas {
		if area.ID == areaID {
			return i
		}
	}
	return -1
}

// DefaultAreaID 默认地区 ID
const DefaultAreaID = "JP13" // 東京
