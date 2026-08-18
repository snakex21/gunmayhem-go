timer = 0;
disabled = false;
randgun = Math.round(this._rotation);
_rotation = 0;
originx = _X;
originy = _Y;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(_root.gamemode == 2 || _root.gamemode == 4 || _root.gamemode == 5 || _root.tournament || !_root.powerON)
      {
         _X = -300;
         _Y = 0;
         _alpha = 0;
         this.swapDepths(1);
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
      if(!disabled)
      {
         if(_alpha < 100)
         {
            _alpha = _alpha + 5;
         }
         i = 0;
         while(i < _root.activeplayers.length)
         {
            if(_root.activeplayers[i].iszombie)
            {
               break;
            }
            if(this.hitTest(_root.activeplayers[i].frame))
            {
               _root.activeplayers[i].getgun(randgun);
               j = 0;
               while(j < 10)
               {
                  _root.CP("fx_pickup",_X + _parent._x,_Y + _parent._y + 20,0);
                  j++;
               }
               _root.playsound2("bolt2.wav");
               disabled = true;
               _X = -300;
               _Y = 0;
               _alpha = 0;
               break;
            }
            i++;
         }
      }
      else
      {
         timer += 1;
         if(timer >= 400)
         {
            gotoAndPlay(random(40) + 10);
            timer = 0;
            disabled = false;
            _X = originx;
            _Y = originy;
         }
      }
   }
};
