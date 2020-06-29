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

COMMENT ON COLUMN IMAGE.IMAGE_FILE_PATH IS '���� � ����� �������� ��������';

COMMENT ON COLUMN IMAGE.IMAGE_STRING IS '�������� �������� � base64';

COMMENT ON COLUMN IMAGE.X0_LAT_GR IS '�������������� ������ ������ �������� ���� ��������';

COMMENT ON COLUMN IMAGE.Y0_LON_GR IS '�������������� ������� ������ �������� ���� ��������';

COMMENT ON COLUMN IMAGE.X_PIXEL_DEG IS '���-�� �������� ������ � �������';

COMMENT ON COLUMN IMAGE.Y_PIXEL_DEG IS '���-�� �������� ������ � �������';

/
COMMIT;
/

