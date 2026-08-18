vx = 20 * asdf;
vy = -6;
rotationspeed = 15 * asdf;
_xscale = _xscale * asdf;
facing = asdf;
hitsomething = false;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _X = _X + vx;
      _Y = _Y + vy;
      _rotation = _rotation + rotationspeed;
      vy += _root.gravity * 1.3;
      if(!hitsomething)
      {
         i = 0;
         while(i < _root.activeplayers.length)
         {
            if(_root.activeplayers[i].frame.hitTest(_X,_Y,true))
            {
               _root.activeplayers[i].vx += 20 * facing;
               _root.activeplayers[i].hitnumber += 1;
               _root.activeplayers[i].hittimer = 0;
               qwer = _root.activeplayers[i].hitnumber;
               j = 0;
               while(j < 4)
               {
                  _root.CP("fx_blood",_X,_Y,0,0);
                  j++;
               }
               _root.CP("fx_bloodstain",_X + vx,_Y,0,0);
               _root.CP("fx_combo",_X + vx,_Y - 50,0,-6);
               tagplayer(_root.activeplayers[i]);
               vx *= -0.5;
               _X = _X + vx;
               hitsomething = true;
            }
            i++;
         }
      }
      if(_Y >= 900 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
