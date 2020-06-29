DROP TABLE IMAGE;
/
COMMIT;
/

CREATE TABLE IMAGE(
    IMAGE_FILE_PATH	NVARCHAR2(300),
	IMAGE_STRING	BLOB,
    X0_LAT_GR       NUMBER,
    Y0_LON_GR       NUMBER,
    X_PIXEL_DEG     NUMBER,
    Y_PIXEL_DEG     NUMBER
);

COMMENT ON COLUMN IMAGE.IMAGE_FILE_PATH IS 'Путь к файлу картинки подложки';

COMMENT ON COLUMN IMAGE.IMAGE_STRING IS 'Картинка подложки в base64';

COMMENT ON COLUMN IMAGE.X0_LAT_GR IS 'Географическая широта левого верхнего угла картинки';

COMMENT ON COLUMN IMAGE.Y0_LON_GR IS 'Географическая долгота левого верхнего угла картинки';

COMMENT ON COLUMN IMAGE.X_PIXEL_DEG IS 'Кол-во градусов широты в пикселе';

COMMENT ON COLUMN IMAGE.Y_PIXEL_DEG IS 'Кол-во градусов доготы в пикселе';

/
COMMIT;
/

