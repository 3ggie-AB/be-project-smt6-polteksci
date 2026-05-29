package seeder

import (
	"log"
	"network-monitor/internal/config"
	"network-monitor/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Run(db *gorm.DB, cfg *config.Config) {
	log.Println("🌱 Menjalankan seeder...")

	seedPermissions(db)
	seedRoles(db)
	seedUsers(db, cfg)
	seedDevices(db)
	seedFeedbacks(db)

	log.Println("✅ Seeder selesai")
}

func seedPermissions(db *gorm.DB) {
	permissions := []models.Permission{
		{Name: "users:read", Description: "Lihat daftar pengguna", Module: "users"},
		{Name: "users:write", Description: "Buat dan edit pengguna", Module: "users"},
		{Name: "users:delete", Description: "Hapus pengguna", Module: "users"},
		{Name: "roles:read", Description: "Lihat daftar role", Module: "roles"},
		{Name: "roles:write", Description: "Kelola role dan permission", Module: "roles"},
		{Name: "monitoring:read", Description: "Lihat hasil monitoring", Module: "monitoring"},
		{Name: "monitoring:write", Description: "Jalankan ping dan SNMP", Module: "monitoring"},
		{Name: "devices:read", Description: "Lihat daftar perangkat", Module: "devices"},
		{Name: "devices:write", Description: "Tambah dan edit perangkat", Module: "devices"},
		{Name: "devices:delete", Description: "Hapus perangkat", Module: "devices"},
		{Name: "feedback:read", Description: "Lihat semua feedback", Module: "feedback"},
		{Name: "feedback:write", Description: "Buat dan edit feedback", Module: "feedback"},
		{Name: "feedback:respond", Description: "Balas dan kelola feedback", Module: "feedback"},
		{Name: "feedback:delete", Description: "Hapus feedback", Module: "feedback"},
	}

	for _, p := range permissions {
		db.Where(models.Permission{Name: p.Name}).FirstOrCreate(&p)
	}
}

func seedRoles(db *gorm.DB) {
	type roleData struct {
		name        string
		displayName string
		description string
		permissions []string
	}

	roles := []roleData{
		{
			name:        "atasan",
			displayName: "Atasan / Manajer",
			description: "Akses penuh ke seluruh sistem, manajemen user dan laporan",
			permissions: []string{
				"users:read", "users:write", "users:delete",
				"roles:read", "roles:write",
				"monitoring:read", "monitoring:write",
				"devices:read", "devices:write", "devices:delete",
				"feedback:read", "feedback:write", "feedback:respond", "feedback:delete",
			},
		},
		{
			name:        "teknisi_it",
			displayName: "Teknisi IT",
			description: "Akses monitoring jaringan, manajemen perangkat, dan respon feedback teknis",
			permissions: []string{
				"users:read",
				"monitoring:read", "monitoring:write",
				"devices:read", "devices:write",
				"feedback:read", "feedback:respond",
			},
		},
		{
			name:        "staff",
			displayName: "Staff",
			description: "Lihat monitoring dan kelola feedback departemen",
			permissions: []string{
				"monitoring:read",
				"devices:read",
				"feedback:read", "feedback:write",
			},
		},
		{
			name:        "karyawan",
			displayName: "Karyawan",
			description: "Buat dan lihat feedback sendiri",
			permissions: []string{
				"feedback:write",
			},
		},
	}

	for _, rd := range roles {
		var role models.Role
		result := db.Where("name = ?", rd.name).First(&role)

		if result.Error != nil {
			role = models.Role{
				Name:        rd.name,
				DisplayName: rd.displayName,
				Description: rd.description,
			}
			db.Create(&role)
		}

		var perms []models.Permission
		db.Where("name IN ?", rd.permissions).Find(&perms)
		db.Model(&role).Association("Permissions").Replace(perms)
	}
}

func seedUsers(db *gorm.DB, cfg *config.Config) {
	type userData struct {
		name       string
		email      string
		password   string
		roleName   string
		department string
		phone      string
	}

	users := []userData{
		{
			name:       cfg.AdminName,
			email:      cfg.AdminEmail,
			password:   cfg.AdminPassword,
			roleName:   "atasan",
			department: "Manajemen",
			phone:      "08111000001",
		},
		{
			name:       cfg.TeknisiName,
			email:      cfg.TeknisiEmail,
			password:   cfg.TeknisiPassword,
			roleName:   "teknisi_it",
			department: "IT Department",
			phone:      "08111000002",
		},
		{
			name:       cfg.StaffName,
			email:      cfg.StaffEmail,
			password:   cfg.StaffPassword,
			roleName:   "staff",
			department: "Administrasi",
			phone:      "08111000003",
		},
		{
			name:       cfg.KaryawanName,
			email:      cfg.KaryawanEmail,
			password:   cfg.KaryawanPassword,
			roleName:   "karyawan",
			department: "Operasional",
			phone:      "08111000004",
		},
	}

	for _, ud := range users {
		var existing models.User
		if err := db.Where("email = ?", ud.email).First(&existing).Error; err == nil {
			continue
		}

		var role models.Role
		db.Where("name = ?", ud.roleName).First(&role)

		hashed, err := bcrypt.GenerateFromPassword([]byte(ud.password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("⚠️  Gagal hash password untuk %s: %v", ud.email, err)
			continue
		}

		user := models.User{
			Name:       ud.name,
			Email:      ud.email,
			Password:   string(hashed),
			RoleID:     role.ID,
			Department: ud.department,
			Phone:      ud.phone,
			IsActive:   true,
		}
		if err := db.Create(&user).Error; err != nil {
			log.Printf("⚠️  Gagal buat user %s: %v", ud.email, err)
		} else {
			log.Printf("   ✓ User dibuat: %s (%s)", ud.name, ud.email)
		}
	}
}

func seedDevices(db *gorm.DB) {
	devices := []models.Device{
		{
			Name:          "Router Utama",
			IPAddress:     "192.168.1.1",
			Type:          "router",
			Location:      "Server Room Lt. 1",
			Description:   "Router utama perusahaan, gateway ke internet",
			SNMPCommunity: "public",
			SNMPVersion:   "2c",
			SNMPPort:      161,
			IsActive:      true,
			CreatedByID:   1,
		},
		{
			Name:          "Switch Core",
			IPAddress:     "192.168.1.254",
			Type:          "switch",
			Location:      "Server Room Lt. 1",
			Description:   "Core switch utama yang menghubungkan semua segmen jaringan",
			SNMPCommunity: "public",
			SNMPVersion:   "2c",
			SNMPPort:      161,
			IsActive:      true,
			CreatedByID:   1,
		},
		{
			Name:          "Server Web",
			IPAddress:     "192.168.1.10",
			Type:          "server",
			Location:      "Server Room Lt. 1",
			Description:   "Server web produksi aplikasi perusahaan",
			SNMPCommunity: "private",
			SNMPVersion:   "2c",
			SNMPPort:      161,
			IsActive:      true,
			CreatedByID:   1,
		},
		{
			Name:          "Server Database",
			IPAddress:     "192.168.1.11",
			Type:          "server",
			Location:      "Server Room Lt. 1",
			Description:   "Server database MySQL dan ClickHouse",
			SNMPCommunity: "private",
			SNMPVersion:   "2c",
			SNMPPort:      161,
			IsActive:      true,
			CreatedByID:   1,
		},
		{
			Name:          "Firewall UTM",
			IPAddress:     "192.168.1.2",
			Type:          "firewall",
			Location:      "Server Room Lt. 1",
			Description:   "Unified Threat Management, keamanan jaringan perimeter",
			SNMPCommunity: "public",
			SNMPVersion:   "2c",
			SNMPPort:      161,
			IsActive:      true,
			CreatedByID:   1,
		},
		{
			Name:          "Access Point Lantai 2",
			IPAddress:     "192.168.2.1",
			Type:          "access_point",
			Location:      "Lantai 2",
			Description:   "Wireless access point untuk area Lantai 2",
			SNMPCommunity: "public",
			SNMPVersion:   "2c",
			SNMPPort:      161,
			IsActive:      true,
			CreatedByID:   2,
		},
		{
			Name:          "Google DNS",
			IPAddress:     "8.8.8.8",
			Type:          "other",
			Location:      "External",
			Description:   "Google Public DNS — digunakan untuk test konektivitas internet",
			SNMPCommunity: "public",
			SNMPVersion:   "2c",
			SNMPPort:      161,
			IsActive:      true,
			CreatedByID:   2,
		},
	}

	for _, d := range devices {
		db.Where(models.Device{IPAddress: d.IPAddress}).FirstOrCreate(&d)
	}
}

func seedFeedbacks(db *gorm.DB) {
	feedbacks := []models.Feedback{
		{
			Title:       "Internet lambat di Lantai 2",
			Description: "Sejak kemarin sore, koneksi internet di Lantai 2 sangat lambat. Banyak karyawan yang tidak bisa bekerja dengan normal. Sudah dicoba restart komputer tapi tidak ada perubahan.",
			Category:    models.CategoryKeluhan,
			Status:      models.StatusOpen,
			Priority:    models.PriorityHigh,
			CreatedByID: 4,
		},
		{
			Title:       "Printer Jaringan Tidak Terdeteksi",
			Description: "Printer jaringan di ruang Administrasi tidak bisa diakses sejak pagi. Sudah dicoba dari beberapa komputer tapi semua gagal. Ini menghambat pekerjaan administrasi.",
			Category:    models.CategoryKeluhan,
			Status:      models.StatusInProgress,
			Priority:    models.PriorityMedium,
			CreatedByID: 3,
			AssignedToID: func() *uint { id := uint(2); return &id }(),
		},
		{
			Title:       "Usulan: Upgrade Bandwidth Internet",
			Description: "Seiring bertambahnya karyawan, bandwidth internet saat ini sudah tidak mencukupi, terutama pada jam kerja puncak. Mohon pertimbangkan untuk upgrade ke paket yang lebih besar.",
			Category:    models.CategorySaran,
			Status:      models.StatusOpen,
			Priority:    models.PriorityLow,
			CreatedByID: 3,
		},
		{
			Title:       "Email Server Down",
			Description: "Email perusahaan tidak bisa diakses sejak pukul 08.00. Pesan error: 'Connection timed out'. Ini sangat mengganggu komunikasi dengan klien.",
			Category:    models.CategoryInsiden,
			Status:      models.StatusResolved,
			Priority:    models.PriorityCritical,
			CreatedByID: 4,
			AssignedToID: func() *uint { id := uint(2); return &id }(),
			Response:    "Email server sudah diperbaiki. Masalah disebabkan oleh disk yang penuh. Sudah dilakukan pembersihan log dan kapasitas disk ditambah. Mohon maaf atas ketidaknyamanannya.",
			RespondedByID: func() *uint { id := uint(2); return &id }(),
		},
		{
			Title:       "Pertanyaan: Cara Akses VPN dari Rumah",
			Description: "Mohon bantuannya, bagaimana cara mengakses VPN perusahaan dari laptop pribadi saat WFH? Sudah mencoba mengikuti panduan lama tapi tidak bisa connect.",
			Category:    models.CategoryPertanyaan,
			Status:      models.StatusResolved,
			Priority:    models.PriorityLow,
			CreatedByID: 4,
			AssignedToID: func() *uint { id := uint(2); return &id }(),
			Response:    "Panduan VPN sudah diperbarui dan dikirim ke email Anda. Pastikan menggunakan aplikasi OpenVPN versi terbaru dan gunakan kredensial yang diberikan oleh tim IT.",
			RespondedByID: func() *uint { id := uint(2); return &id }(),
		},
		{
			Title:       "CCTV Lobby Tidak Merekam",
			Description: "CCTV di area lobby tidak merekam sejak 3 hari lalu. Storage NVR mungkin penuh. Mohon segera ditindaklanjuti untuk keamanan kantor.",
			Category:    models.CategoryKeluhan,
			Status:      models.StatusInProgress,
			Priority:    models.PriorityHigh,
			CreatedByID: 3,
			AssignedToID: func() *uint { id := uint(2); return &id }(),
		},
	}

	var count int64
	db.Model(&models.Feedback{}).Count(&count)
	if count == 0 {
		for _, f := range feedbacks {
			db.Create(&f)
		}
	}
}
