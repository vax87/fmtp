create or replace PACKAGE PARKING_PKG AS 
    -- стоянка
    TYPE PARKING_INFO IS RECORD
    (        
        M_AIRPORT_CODE          PARKINGS.AIRPORT_ICAO_CODE%TYPE := '',
        M_PARKING_NAME          PARKINGS.PARKING_NAME%TYPE := '', 
        M_UNITED_PARKING_NAME   PARKINGS.UNITED_PARKING_NAME%TYPE := '',
        M_X_POS                 PARKINGS.X_POS%TYPE := 0,
        M_Y_POS                 PARKINGS.Y_POS%TYPE := 0,
        M_X_POS_AMMEND          PARKINGS.X_POS_AMMEND%TYPE := 0,
        M_Y_POS_AMMEND          PARKINGS.Y_POS_AMMEND%TYPE := 0,
        M_ARCFT_TYPES           PARKINGS.ARCFT_TYPES%TYPE := '',
        M_CUR_STATE             PARKINGS.CUR_STATE%TYPE := '',
        M_FLIGHT_NUM            PARKINGS.FLIGHT_NUM%TYPE := '',
        M_SIDE_NUM              PARKINGS.SIDE_NUM%TYPE := '',
        M_INFO                  PARKINGS.INFO%TYPE := '',
        M_SECTOR_CODE           PARKINGS.SECTOR_CODE%TYPE := ''
    );
    
    
    -- параметры картинки
    TYPE IMAGE_INFO IS RECORD
    (
        M_IMAGE_STRING  IMAGE.IMAGE_STRING%TYPE,
        M_X0_LAT_GR     IMAGE.X0_LAT_GR%TYPE,
        M_Y0_LON_GR     IMAGE.Y0_LON_GR%TYPE,
        M_X_PIXEL_DEG   IMAGE.X_PIXEL_DEG%TYPE,
        M_Y_PIXEL_DEG   IMAGE.Y_PIXEL_DEG%TYPE
    );
          

    --инициализация стоянки при получении данных от АНИ
    PROCEDURE  INIT_PARKING_ANI( PRK IN PARKING_INFO );    
    
    --инициализация стоянки при получении данных от WEB
    PROCEDURE  INIT_PARKING_WEB( PRK IN PARKING_INFO );
    
    --изменение состояния стоянки в клиенте
    PROCEDURE  UPDATE_PARKING_STATE( PRK IN PARKING_INFO );   
        
    --изменение состояния стоянки в клиенте
    PROCEDURE  UPDATE_IMAGE_PARAMS( IMG IN IMAGE_INFO );   
END;
/

create or replace PACKAGE BODY PARKING_PKG AS 


    --инициализация стоянки при получении данных от АНИ
    PROCEDURE INIT_PARKING_ANI( PRK IN PARKING_INFO ) 
    AS
        CUR_PRK_ID PARKINGS.ID%TYPE;
    BEGIN
    
        SELECT NVL(max(ID), 0) INTO CUR_PRK_ID FROM PARKINGS WHERE AIRPORT_ICAO_CODE = PRK.M_AIRPORT_CODE AND PARKING_NAME = PRK.M_PARKING_NAME;

        IF CUR_PRK_ID = 0 THEN
            INSERT INTO PARKINGS (
                AIRPORT_ICAO_CODE,
                PARKING_NAME,
                UNITED_PARKING_NAME,
                X_POS,
                Y_POS,
                X_POS_AMMEND,
                Y_POS_AMMEND,
                ARCFT_TYPES
                )
            VALUES (
                PRK.M_AIRPORT_CODE,
                PRK.M_PARKING_NAME,
                PRK.M_UNITED_PARKING_NAME,
                PRK.M_X_POS,
                PRK.M_Y_POS,
                PRK.M_X_POS_AMMEND,
                PRK.M_Y_POS_AMMEND,
                PRK.M_ARCFT_TYPES
            );
        ELSE
            UPDATE PARKINGS SET 
                X_POS = PRK.M_X_POS, 
                Y_POS = PRK.M_Y_POS,
                ARCFT_TYPES = PRK.M_ARCFT_TYPES                
            WHERE ID = CUR_PRK_ID;                              
        END IF;    
    END INIT_PARKING_ANI;


    --инициализация стоянки при получении данных от WEB
    PROCEDURE  INIT_PARKING_WEB( PRK IN PARKING_INFO )
    AS
        CUR_PRK_ID PARKINGS.ID%TYPE;
    BEGIN 
         SELECT NVL(max(ID), 0) INTO CUR_PRK_ID FROM PARKINGS WHERE AIRPORT_ICAO_CODE = PRK.M_AIRPORT_CODE AND PARKING_NAME = PRK.M_PARKING_NAME;

        IF CUR_PRK_ID != 0 THEN
            UPDATE PARKINGS SET 
                X_POS_AMMEND = PRK.M_X_POS_AMMEND, 
                Y_POS_AMMEND = PRK.M_Y_POS_AMMEND,
                UNITED_PARKING_NAME = PRK.M_UNITED_PARKING_NAME                
            WHERE ID = CUR_PRK_ID;                              
        END IF;    
    END INIT_PARKING_WEB;


    --изменение состояния стоянки в клиенте
    PROCEDURE UPDATE_PARKING_STATE( PRK IN PARKING_INFO )
    AS
        CUR_PRK_ID PARKINGS.ID%TYPE;
    BEGIN
        SELECT NVL(max(ID), 0) INTO CUR_PRK_ID FROM PARKINGS WHERE AIRPORT_ICAO_CODE = PRK.M_AIRPORT_CODE AND PARKING_NAME = PRK.M_PARKING_NAME;

        IF CUR_PRK_ID != 0 THEN
            UPDATE PARKINGS
            SET
                CUR_STATE         = PRK.M_CUR_STATE,
                FLIGHT_NUM        = PRK.M_FLIGHT_NUM,
                SIDE_NUM          = PRK.M_SIDE_NUM,
                INFO              = PRK.M_INFO,
                SECTOR_CODE       = PRK.M_SECTOR_CODE
            WHERE ID = CUR_PRK_ID;
        ELSE
            INSERT INTO PARKINGS (
                AIRPORT_ICAO_CODE,
                PARKING_NAME,
                UNITED_PARKING_NAME,
                X_POS,
                Y_POS,
                ARCFT_TYPES,
                CUR_STATE,
                FLIGHT_NUM,
                SIDE_NUM,
                INFO,
                SECTOR_CODE
                )
            VALUES (
                PRK.M_AIRPORT_CODE,
                PRK.M_PARKING_NAME,
                PRK.M_UNITED_PARKING_NAME,
                PRK.M_X_POS,
                PRK.M_Y_POS,
                PRK.M_ARCFT_TYPES,
                PRK.M_CUR_STATE,
                PRK.M_FLIGHT_NUM,
                PRK.M_SIDE_NUM,
                PRK.M_INFO,
                PRK.M_SECTOR_CODE
            );
        END IF;
    END UPDATE_PARKING_STATE;   
    
    
    --изменение состояния стоянки в клиенте
    PROCEDURE  UPDATE_IMAGE_PARAMS( IMG IN IMAGE_INFO )
    AS   
      CNT_ROWS NUMBER;
    BEGIN
      SELECT COUNT(*) INTO CNT_ROWS FROM IMAGE ;
        
      IF CUR_PRK_ID != 0 THEN
        UPDATE IMAGE SET
          IMAGE_STRING  = IMG.M_IMAGE_STRING,
          X0_LAT_GR     = IMG.M_X0_LAT_GR,
          Y0_LON_GR     = IMG.M_Y0_LON_GR,
          X_PIXEL_DEG   = IMG.M_X_PIXEL_DEG,
          Y_PIXEL_DEG   = IMG.M_X_PIXEL_DEG;
      ELSE
        INSERT INTO PARKINGS 
            VALUES (
              IMG.M_IMAGE_STRING,
              IMG.M_X0_LAT_GR,
              IMG.M_Y0_LON_GR,
              IMG.M_X_PIXEL_DEG,
              IMG.M_X_PIXEL_DEG
            );
            
    END UPDATE_IMAGE_PARAMS;
END;
/
COMMIT;
/
