stop();
pickedup = false;
powerupnumber = random(7);
if(powerupnumber == 4 && _root.double || powerupnumber == 4 && _root.campaignmode)
{
   powerupnumber = 2;
}
_root.cratearray[_root.cratearray.length] = this;
this.onEnterFrame = function()
{
   if(_currentframe >= 20 && !pickedup)
   {
      i = 0;
      loop0:
      for(; i < _root.activeplayers.length; i++)
      {
         if(_root.activeplayers[i].iszombie)
         {
            break;
         }
         if(!_root.activeplayers[i].frame.hitTest(frame))
         {
            continue;
         }
         j = 0;
         while(j < 10)
         {
            _root.CP("fx_pickup",_X,_Y + 40,0);
            j++;
         }
         _root.CP("fx_powerupname",_X,_Y,0,powerupnumber + 1);
         pickedup = true;
         _root.playsound("emp.wav");
         _root.pgsdata[_root.activeplayers[i].PLAYERNUMBER - 1][6] += 1;
         if(_root.activeplayers[i]._name == "double")
         {
            continue;
         }
         switch(powerupnumber)
         {
            case 0:
               _root.activeplayers[i].lives = parseInt(_root.activeplayers[i].lives) + 1;
               _root.activeplayers[i].lifebling();
               break loop0;
            case 1:
               _root.activeplayers[i].invisibletime = 200;
               break loop0;
            case 2:
               _root.activeplayers[i].shieldtime = 300;
               break loop0;
            case 3:
               _root.activeplayers[i].jetfuel = 100;
               break loop0;
            case 4:
               _root.activeplayers[i].spawnfriend();
               break loop0;
            case 5:
               _root.activeplayers[i].speedtime = 300;
               break loop0;
            case 6:
               _root.activeplayers[i].minitime = 400;
               break loop0;
            default:
               break loop0;
         }
      }
   }
   if(pickedup && stuff._alpha != 0)
   {
      stuff._alpha = 0;
      aura._alpha = 0;
   }
   if(pickedup && flashystuff._currentframe == flashystuff._totalframes)
   {
      j = 0;
      while(j < _root.cratearray.length)
      {
         if(_root.cratearray[j] == this)
         {
            _root.cratearray.splice(j,1);
         }
         j++;
      }
      removeMovieClip(this);
      delete this.onEnterFrame;
   }
   if(_root.deleteeverything || _root.gamewin)
   {
      removeMovieClip(this);
      delete this.onEnterFrame;
   }
};
