falling = true;
vy = 10;
stop();
if(_X > _root.ground._x + _root.ground.platform._x + _root.ground.platform._width / 2)
{
   _xscale = _xscale * -1;
}
_root.cratearray[_root.cratearray.length] = this;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(falling)
      {
         if(vy < 18)
         {
            vy += 1;
         }
         _Y = _Y + vy;
         if(_root.ground.platform.hitTest(_X,_Y,true))
         {
            falling = false;
            _Y = _Y - vy;
            i = 1;
            while(i <= 5)
            {
               if(_root.ground.platform.hitTest(_X,_Y + i * (vy / 5),true))
               {
                  _Y = _Y + vy / 5 * (i - 1);
                  play();
                  break;
               }
               i++;
            }
         }
      }
      i = 0;
      while(i < _root.activeplayers.length)
      {
         if(_root.activeplayers[i].iszombie)
         {
            break;
         }
         if(_root.campaignlevel == 10 && _root.activeplayers[i].AI)
         {
            break;
         }
         if(this.hitTest(_root.activeplayers[i].frame))
         {
            do
            {
               randgun = random(57) + 10;
            }
            while(_root.savedata3.data.gunarray[randgun - 10] == false);
            if(_root.activeplayers[i].perknumber == 4 && random(3) == 0)
            {
               randgun = 55;
            }
            if(_root.gamemode == 2)
            {
               randgun = 9;
            }
            if(randgun == 26 && _root.activeplayers[i].AI)
            {
               randgun = 55;
            }
            _root.activeplayers[i].getgun(randgun);
            _root.pgsdata[_root.activeplayers[i].PLAYERNUMBER - 1][5] += 1;
            i = 0;
            while(i < 5)
            {
               _root.CP("fx_package",_X,_Y,0);
               i++;
            }
            _root.playsound2("bolt2.wav");
            removeMovieClip(this);
            delete this.onEnterFrame;
            j = 0;
            while(j < _root.cratearray.length)
            {
               if(_root.cratearray[j] == this)
               {
                  _root.cratearray.splice(j,1);
               }
               j++;
            }
            break;
         }
         i++;
      }
      if(_root.deleteeverything || _root.gamewin)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
